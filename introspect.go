package dbus

import (
	"xml"
	"os"
	"bytes"
	"strings"
)

type annotationData struct {
	Name  string "attr"
	Value string "attr"
}

type argData struct {
	Name      string "attr"
	Type      string "attr"
	Direction string "attr"
}

type methodData struct {
	Name       string "attr"
	Arg        []argData
	Annotation annotationData
}

type signalData struct {
	Name string "attr"
	Arg  []argData
}

type interfaceData struct {
	Name   string "attr"
	Method []methodData
	Signal []signalData
}

type introspect struct {
	Name      string "attr"
	Interface []interfaceData
	Node      []*introspect
}

type Introspect interface {
	GetInterfaceData(name string) InterfaceData
}

type InterfaceData interface {
	GetMethodData(name string) MethodData
	GetSignalData(name string) SignalData
	GetName() string
}

type MethodData interface {
	GetName() string
	GetInSignature() string
	GetOutSignature() string
}

type SignalData interface {
	GetSignature() string
}

func NewIntrospect(xmlIntro string) (Introspect, os.Error) {
	intro := new(introspect)
	buff := bytes.NewBuffer(strings.Bytes(xmlIntro))
	err := xml.Unmarshal(buff, intro)
	if err != nil {
		return nil, err
	}

	return intro, nil
}

func (p introspect) GetInterfaceData(name string) InterfaceData {
	for _, v := range p.Interface {
		if v.Name == name {
			return v
		}
	}
	return nil
}

func (p interfaceData) GetMethodData(name string) MethodData {
	for _, v := range p.Method {
		if v.GetName() == name {
			return v
		}
	}
	return nil
}

func (p interfaceData) GetSignalData(name string) SignalData {
	for _, v := range p.Signal {
		if v.GetName() == name {
			return v
		}
	}
	return nil
}

func (p interfaceData) GetName() string { return p.Name }

func (p methodData) GetInSignature() (sig string) {
	for _, v := range p.Arg {
		if strings.ToUpper(v.Direction) == "IN" {
			sig += v.Type
		}
	}
	return
}

func (p methodData) GetOutSignature() (sig string) {
	for _, v := range p.Arg {
		if strings.ToUpper(v.Direction) == "OUT" {
			sig += v.Type
		}
	}
	return
}

func (p methodData) GetName() string { return p.Name }

func (p signalData) GetSignature() (sig string) {
	for _, v := range p.Arg {
		sig += v.Type
	}
	return
}

func (p signalData) GetName() string { return p.Name }
