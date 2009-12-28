package dbus

import (
	"net"
	"regexp"
	"os"
	"fmt"
	"strings"
	"bytes"
	"reflect"
)

type Connection struct {
	path              string
	uniqName          string
	guid              string
	methodCallReplies map[uint32](func(msg *Message))
	conn              net.Conn
	buffer            *bytes.Buffer
}

type Object struct {
	bus   *Connection
	dest  string
	path  string
	intro Introspect
}

type Interface struct {
	bus   *Connection
	obj   *Object
	name  string
	intro InterfaceData
}

func (p *Connection) Initialize() os.Error {
	p.methodCallReplies = make(map[uint32]func(*Message))
	p.buffer = bytes.NewBuffer([]byte{})
	p._CreateSocket()
	p._Auth()
	go p._RunLoop()
	p._SendHello()
	return nil
}

func (p *Connection) _CreateSocket() os.Error {
	var re *regexp.Regexp
	re, _ = regexp.Compile("^unix:abstract=(.*),guid=(.*)")
	m := re.ExecuteString(p.path)
	if nil != m {
		abPath := p.path[m[2]:m[3]] // get regexp 1st group
		addr, _ := net.ResolveUnixAddr("unix", "\x00"+abPath)
		conn, _ := net.DialUnix("unix", nil, addr)
		p.conn = conn
		return nil
	}

	return nil
}

func (p *Connection) _Auth() os.Error {
	p.conn.Write(strings.Bytes("\x00"))
	p.conn.Write(strings.Bytes("AUTH EXTERNAL " + fmt.Sprintf("%x", fmt.Sprintf("%d", os.Getuid())) + "\r\n"))

	b := make([]byte, 1000)
	p.conn.Read(b)
	retstr := string(b)
	re, _ := regexp.Compile("^OK ([0-9a-fA-F]+)")
	m := re.ExecuteString(retstr)
	if nil != m {
		guid := retstr[m[2]:m[3]]
		p.guid = guid
		p.conn.Write(strings.Bytes("BEGIN\r\n"))
		return nil
	}

	return os.NewError("Auth Failed")
}

func (p *Connection) _MessageReceiver(msgChan chan *Message) {
	for {
		msg, e := p._PopMessage()
		if e == nil {
			msgChan <- msg
			continue // might be another msg in p.buffer
		}
		p._UpdateBuffer()
	}
}

func (p *Connection) _RunLoop() {
	msgChan := make(chan *Message)
	go p._MessageReceiver(msgChan)
	for {
		select {
		case msg := <-msgChan:
			p._MessageDispatch(msg)
		}
	}
}

func (p *Connection) _MessageDispatch(msg *Message) {
	fmt.Printf("%#v\n", msg)
	if msg == nil {
		return
	}

	switch msg.Type {
	case METHOD_RETURN:
		rs := msg.replySerial
		if replyFunc, ok := p.methodCallReplies[rs]; ok {
			replyFunc(msg)
			p.methodCallReplies[rs] = nil, false
		}
	case ERROR:
		fmt.Println("ERROR")
		fmt.Printf("%#v\n", msg)
	}
}

func (p *Connection) _PopMessage() (*Message, os.Error) {
	msg, n, err := _Unmarshal(p.buffer.Bytes())
	if err != nil {
		return nil, err
	}
	p.buffer.Read(make([]byte, n)) // remove first n bytes
	return msg, nil
}

func (p *Connection) _UpdateBuffer() os.Error {
	//	_, e := p.buffer.ReadFrom(p.conn);
	buff := make([]byte, 4096)
	n, e := p.conn.Read(buff)
	p.buffer.Write(buff[0:n])
	return e
}

func (p *Connection) _SendSync(msg *Message, callback func(*Message)) os.Error {
	seri := uint32(msg.serial)
	recvChan := make(chan int)
	p.methodCallReplies[seri] = func(rmsg *Message) {
		callback(rmsg)
		recvChan <- 0
	}

	buff, _ := msg._Marshal()
	fmt.Printf("%q\n", buff)
	p.conn.Write(buff)
	fmt.Println("sync start", len(buff))
	<-recvChan // synchronize
	fmt.Println("sync end")
	return nil
}

func (p *Connection) _SendHello() os.Error {
	fmt.Println("send hello start")
	msg := NewMessage()
	msg.Type = METHOD_CALL
	msg.Path = "/org/freedesktop/DBus"
	msg.Intf = "org.freedesktop.DBus"
	msg.Dest = "org.freedesktop.DBus"
	msg.Member = "Hello"
	p._SendSync(msg, func(reply *Message) { fmt.Println("send hello success") })
	return nil
}

func (p *Connection) GetObject(dest string, path string) *Object {
	msg := NewMessage()
	msg.Type = METHOD_CALL
	msg.Path = path
	msg.Dest = dest
	msg.Intf = "org.freedesktop.DBus.Introspectable"
	msg.Member = "Introspect"

	obj := new(Object)
	obj.bus = p
	obj.path = path
	obj.dest = dest

	p._SendSync(msg, func(reply *Message) {
		if v, ok := reply.Params.At(0).(string); ok {
			if intro, err := NewIntrospect(v); err == nil {
				obj.intro = intro
			}
		}
	})

	return obj
}

func (p *Connection) Interface(obj *Object, name string) *Interface {

	if obj == nil || obj.intro == nil {
		return nil
	}

	intf := new(Interface)
	intf.bus = p
	intf.obj = obj
	intf.name = name

	data := obj.intro.GetInterfaceData(name)
	if nil == data {
		return nil
	}

	intf.intro = data

	return intf
}

func (p *Interface) CallMethod(name string, args ...) os.Error {

	method := p.intro.GetMethodData(name)
	if nil == method {
		return os.NewError("Invalid Method")
	}

	msg := NewMessage()

	v := reflect.NewValue(args).(*reflect.StructValue)
	for i := 0; i < v.NumField(); i++ {
		val := v.Field(i)
		if inter := val.Interface(); inter != nil {
			msg.Params.Push(inter)
		}
	}

	msg.Type = METHOD_CALL
	msg.Path = p.obj.path
	msg.Intf = p.name
	msg.Dest = p.obj.dest
	msg.Member = name
	msg.Sig = method.GetInSignature()

	p.bus._SendSync(msg, func(reply *Message) { fmt.Println("Method Call Comp:", name) })

	return nil
}
