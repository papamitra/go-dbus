
package dbus

import (
	"container/vector";
	"testing";
	"os";
	"fmt";
)

func TestDbus(t *testing.T){
}

func TestConnectionInitialize(t *testing.T){
	con := Connection{path: os.Getenv("DBUS_SESSION_BUS_ADDRESS")};
	e := con.Initialize()

	if e != nil { t.Error("#1 Failed")}

	obj := con.GetObject("org.freedesktop.Notifications", "/org/freedesktop/Notifications")
	fmt.Println(obj.intro)

	inf := con.Interface(obj,"org.freedesktop.Notifications")
	if inf == nil { t.Error("Failed #3")}

	method := inf.intro.GetMethodData("Notify")
	fmt.Println(method)

	inf.CallMethod("Notify", "dbus.go", uint32(0), "info", "test", "test_body", (*vector.Vector)(nil), (*vector.Vector)(nil), int32(2000))

}
