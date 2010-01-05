
package dbus

import (
	"container/vector"
	"fmt"
	"reflect"
	"strings"
)

var typeMap = map[MessageType] string{
  INVALID: "invalid",
	METHOD_CALL: "method_call",
	METHOD_RETURN: "method_return",
  SIGNAL : "signal",
  ERROR : "error",
}
	
type MatchRule struct{
	Type string
	Interface string
	Member string
	Path string
}

func(p *MatchRule) _ToString() string{
	svec := new(vector.StringVector)

	v := reflect.Indirect(reflect.NewValue(p)).(*reflect.StructValue)
	t := v.Type().(*reflect.StructType)
	for i:=0; i<v.NumField(); i++{
		str, ok := v.Field(i).Interface().(string)
		if ok && "" != str{
			svec.Push(fmt.Sprintf("%s='%s'", strings.ToLower(t.Field(i).Name), str))
		}	
	}
	
	return strings.Join(svec.Data(),",")
}

func(p *MatchRule) _Match(msg *Message) bool{
	if p.Type != "" && p.Type != typeMap[msg.Type]{ return false}
	if p.Interface != "" && p.Interface != msg.Iface { return false}
	if p.Member != "" && p.Member != msg.Member { return false}
	if p.Path != "" && p.Path != msg.Path { return false}
	return true
}

	