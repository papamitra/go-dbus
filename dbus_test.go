
package dbus

import (
	"testing"
	"fmt"
)

func TestDbus(t *testing.T){
	con,_ := NewSessionBus()
	e := con.Initialize()

	if e != nil { t.Error("#1 Failed")}

	obj := con.GetObject("org.freedesktop.Notifications", "/org/freedesktop/Notifications")

	inf := con.Interface(obj,"org.freedesktop.Notifications")
	if inf == nil { t.Error("Failed #3")}

	ret,_ := con.CallMethod(inf, "Notify", "dbus.go", uint32(0), "info", "test", "test_body", []string{}, map[uint32] interface{}{}, int32(2000))
	fmt.Println(ret)

	
}
