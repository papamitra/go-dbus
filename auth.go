
package dbus

import(
	"strings"
	"container/list"
	"fmt"
	"os"
	"net"
)

var(
	ErrAuthUnknownCommand = os.NewError("UnknowAuthCommand")
	ErrAuthFailed = os.NewError("AuthenticationFailed")
)

type Authenticator interface{
	Mechanism() string;
	Authenticate() string;
}

type AuthExternal struct{
}

func(p *AuthExternal) Mechanism() string{ return "EXTERNAL"}
func(p *AuthExternal) Authenticate() string{
	return fmt.Sprintf("%x", fmt.Sprintf("%d", os.Getuid()))
}

type authStatus int
const (
	STARTING = iota
	WAITING_FOR_DATA
	WAITING_FOR_OK
	WAITING_FOR_REJECT
	AUTH_CONTINUE
	AUTH_OK
	AUTH_ERROR
	AUTHENTICATED
	AUTH_NEXT
)

type authState struct{
	status authStatus
	auth Authenticator
	authList list.List
	conn net.Conn
}

func(p *authState) AddAuthenticator(auth Authenticator){
	p.authList.PushBack(auth)
}

func(p *authState) _NextAuthenticator(){
	if p.authList.Len() == 0{
		p.auth = nil
		return
	}

	p.auth,_ = p.authList.Front().Value.(Authenticator)
	p.authList.Remove(p.authList.Front())
	msg := strings.Join([]string{"AUTH", p.auth.Mechanism(), p.auth.Authenticate()}, " ")
	p._Send(msg)
}

func(p *authState) _NextMessage() []string{
	b := make([]byte, 4096)
	p.conn.Read(b)
	retstr := string(b)
	return strings.Split(strings.TrimSpace(retstr), " ", 0)
}

func(p *authState) _Send(msg string){
	p.conn.Write(strings.Bytes(msg + "\r\n"));
}

func(p *authState) Authenticate(conn net.Conn) os.Error{
	p.conn = conn
	p.conn.Write(strings.Bytes("\x00"))
	p._NextAuthenticator()
	p.status = STARTING
	for ;p.status != AUTHENTICATED;{
		if nil == p.auth { return ErrAuthFailed}
		if err := p._NextState(); err != nil{ return err}
	}
	return nil
}

func(p *authState) _NextState() (err os.Error){
	nextMsg := p._NextMessage()
	
	if STARTING == p.status {
		switch nextMsg[0]{
		case "CONTINUE":
			p.status = WAITING_FOR_DATA
		case "OK":
			p.status = WAITING_FOR_OK
		}
	}

	switch p.status{
	case WAITING_FOR_DATA:
		err = p._WaitingForData(nextMsg)
	case WAITING_FOR_OK:
		err = p._WaitingForOK(nextMsg)
	case WAITING_FOR_REJECT:
		err = p._WaitingForReject(nextMsg)
	}

	return;
}

func(p *authState) _WaitingForData(msg []string) os.Error{
	switch msg[0]{
	case "DATA":
		return ErrAuthUnknownCommand
	case "REJECTED":
		p._NextAuthenticator()
		p.status = WAITING_FOR_DATA
	case "OK":
		p._Send("BEGIN")
		p.status = AUTHENTICATED
	default:
		p._Send("ERROR")
		p.status = WAITING_FOR_DATA
	}
	return nil
}

func(p *authState) _WaitingForOK(msg []string) os.Error{
	switch msg[0]{
	case "OK":
		p._Send("BEGIN")
		p.status = AUTHENTICATED
	case "REJECT":
		p._NextAuthenticator()
		p.status = WAITING_FOR_DATA
	case "DATA", "ERROR":
		p._Send("CANCEL")
		p.status = WAITING_FOR_REJECT
	default:
		p._Send("ERROR")
		p.status = WAITING_FOR_OK
	}

	return nil
}

func(p *authState) _WaitingForReject(msg []string) os.Error{
	switch msg[0]{
	case "REJECT":
		p._NextAuthenticator()
		p.status = WAITING_FOR_OK
	default:
		return ErrAuthUnknownCommand
	}
	return nil
}
