
package dbus

import(
	"testing"
)

func TestToString(t *testing.T){
	verifyStr := "type='signal',interface='org.freedesktop.DBus',member='Foo',path='/bar/foo'"

	mr := MatchRule{
	  Type:"signal",
	  Interface:"org.freedesktop.DBus",
  	Member:"Foo",
	  Path:"/bar/foo"}

	if mr._ToString() != verifyStr { t.Error("#1 Failed")}
}
