package dbus

import (
	"testing"
)

var introStr = `
        <!DOCTYPE node PUBLIC "-//freedesktop//DTD D-BUS Object Introspection 1.0//EN"
         "http://www.freedesktop.org/standards/dbus/1.0/introspect.dtd">
        <node name="/org/freedesktop/sample_object">
          <interface name="org.freedesktop.SampleInterface">
            <method name="Frobate">
              <arg name="foo" type="i" direction="in"/>
              <arg name="bar" type="s" direction="out"/>
              <arg name="baz" type="a{us}" direction="out"/>
              <annotation name="org.freedesktop.DBus.Deprecated" value="true"/>
            </method>
            <method name="Bazify">
              <arg name="bar" type="(iiu)" direction="in"/>
              <arg name="bar" type="v" direction="out"/>
            </method>
            <method name="Mogrify">
              <arg name="bar" type="(iiav)" direction="in"/>
            </method>
            <signal name="Changed">
              <arg name="new_value" type="b"/>
            </signal>
            <property name="Bar" type="y" access="readwrite"/>
          </interface>
          <node name="child_of_sample_object"/>
          <node name="another_child_of_sample_object"/>
       </node>
`

func TestIntrospect(t *testing.T) {
	intro, e := NewIntrospect(introStr)
	if e != nil {
		t.Error("Failed #1-1")
	}
	if intro == nil {
		t.Error("Failed #1-2")
	}

	intf := intro.GetInterfaceData("org.freedesktop.SampleInterface")
	if intf == nil {
		t.Error("Failed #2-1")
	}
	if intf.GetName() != "org.freedesktop.SampleInterface" {
		t.Error("Failed #2-2")
	}

	meth := intf.GetMethodData("Frobate")
	if meth == nil {
		t.Error("Failed #3-1")
	}
	if meth != nil && "sa{us}" != meth.GetOutSignature() {
		t.Error("Failed #3-2")
	}

	nilmeth := intf.GetMethodData("Hoo") // unknown method name
	if nilmeth != nil {
		t.Error("Failed #3-3")
	}

	signal := intf.GetSignalData("Changed")
	if signal == nil {
		t.Error("Failed #4-1")
	}
	if signal != nil && "b" != signal.GetSignature() {
		t.Error("Failed #4-2")
	}

	nilsignal := intf.GetSignalData("Hoo") // unknown signal name
	if nilsignal != nil {
		t.Error("Failed #4-3")
	}

}
