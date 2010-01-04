
package dbus

import (
	"container/vector"
	"fmt"
	"reflect"
	"strings"
)

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
