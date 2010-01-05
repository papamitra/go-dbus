package dbus

import (
	"container/vector"
	"os"
	"bytes"
	"sync"
	//"fmt";
)

type MessageType int

const (
	INVALID       = 0
	METHOD_CALL   = 1
	METHOD_RETURN = 2
	ERROR         = 3
	SIGNAL        = 4
)

type MessageFlag int

const (
	NO_REPLY_EXPECTED = 0x1
	NO_AUTO_START     = 0x2
)

type Message struct {
	Type        MessageType
	Flags       MessageFlag
	Protocol    int
	bodyLength  int
	Path        string
	Dest        string
	Iface        string
	Member      string
	Sig         string
	Params      *vector.Vector
	serial      int
	replySerial uint32
	ErrorName   string
	//	Sender;
}

var serialMutex sync.Mutex
var messageSerial = int(0)

func _GetNewSerial() int {
	serialMutex.Lock()
	messageSerial++
	serial := messageSerial
	serialMutex.Unlock()
	return serial
}

func NewMessage() *Message {
	msg := new(Message)

	msg.serial = _GetNewSerial()
	msg.replySerial = 0
	msg.Flags = 0
	msg.Protocol = 1

	msg.Params = new(vector.Vector)

	return msg
}

func (p *Message) _BufferToMessage(buff []byte) (int, os.Error) {
	vec, bufIdx, e := Parse(buff, "yyyyuua(yv)", 0)
	if e != nil {
		return 0, e
	}

	p.Type = MessageType(vec.At(1).(byte))
	p.Flags = MessageFlag(vec.At(2).(byte))
	p.Protocol = int(vec.At(3).(byte))
	p.bodyLength = int(vec.At(4).(uint32))
	p.serial = int(vec.At(5).(uint32))

	for v := range vec.At(6).(*vector.Vector).Iter() {
		t := int(v.(*vector.Vector).At(0).(byte))
		val := v.(*vector.Vector).At(1)

		switch t {
		case 1:
			p.Path = val.(string)
		case 2:
			p.Iface = val.(string)
		case 3:
			p.Member = val.(string)
		case 4:
			p.ErrorName = val.(string)
		case 5:
			p.replySerial = val.(uint32)
		case 6:
			p.Dest = val.(string)
		case 7:
			// FIXME
		case 8:
			p.Sig = val.(string)
		}
	}
	idx := _Align(8, bufIdx)
	if 0 < p.bodyLength {
		vec, idx, _ = Parse(buff, p.Sig, idx)
		p.Params.AppendVector(vec)
	}
	return idx, nil
}

func _Unmarshal(buff []byte) (*Message, int, os.Error) {
	msg := NewMessage()
	idx, e := msg._BufferToMessage(buff)
	if e != nil {
		return nil, 0, e
	}
	return msg, idx, nil
}

func (p *Message) _Marshal() ([]byte, os.Error) {
	buff := bytes.NewBuffer([]byte{})
	_AppendByte(buff, byte('l')) // little Endian
	_AppendByte(buff, byte(p.Type))
	_AppendByte(buff, byte(p.Flags))
	_AppendByte(buff, byte(p.Protocol))

	tmpBuff := bytes.NewBuffer([]byte{})
	_AppendParamsData(tmpBuff, p.Sig, p.Params)
	_AppendUint32(buff, uint32(len(tmpBuff.Bytes())))
	_AppendUint32(buff, uint32(p.serial))

	_AppendArray(buff, 1,
		func(b *bytes.Buffer) {
			if p.Path != "" {
				_AppendAlign(8, b)
				_AppendByte(b, 1) // path
				_AppendByte(b, 1) // signature size
				_AppendByte(b, 'o')
				_AppendByte(b, 0)
				_AppendString(b, p.Path)
			}

			if p.Iface != "" {
				_AppendAlign(8, b)
				_AppendByte(b, 2) // interface
				_AppendByte(b, 1) // signature size
				_AppendByte(b, 's')
				_AppendByte(b, 0)
				_AppendString(b, p.Iface)
			}

			if p.Member != "" {
				_AppendAlign(8, b)
				_AppendByte(b, 3) // member
				_AppendByte(b, 1) // signature size
				_AppendByte(b, 's')
				_AppendByte(b, 0)
				_AppendString(b, p.Member)
			}

			if p.replySerial != 0 {
				_AppendAlign(8, b)
				_AppendByte(b, 5) // reply serial
				_AppendByte(b, 1) // signature size
				_AppendByte(b, 'u')
				_AppendByte(b, 0)
				_AppendUint32(b, uint32(p.replySerial))
			}

			if p.Dest != "" {
				_AppendAlign(8, b)
				_AppendByte(b, 6) // destination
				_AppendByte(b, 1) // signature size
				_AppendByte(b, 's')
				_AppendByte(b, 0)
				_AppendString(b, p.Dest)
			}

			if p.Sig != "" {
				_AppendAlign(8, b)
				_AppendByte(b, 8) // signature
				_AppendByte(b, 1) // signature size
				_AppendByte(b, 'g')
				_AppendByte(b, 0)
				_AppendSignature(b, p.Sig)
			}
		})

	_AppendAlign(8, buff)
	_AppendParamsData(buff, p.Sig, p.Params)

	return buff.Bytes(), nil
}
