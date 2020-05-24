package main

import (
	"github.com/willscott/goturn"
	"net"
	"testing"
	"time"
)

func TestStunUDP(t *testing.T) {
	dialConn, dialError := net.Dial("udp", "127.0.0.1:1443")
	if dialError != nil {
		t.Fail()
		return
	}

	time.Sleep(1*time.Second)
	message, reqError := goturn.NewBindingRequest()
	if reqError != nil {
		t.Fail()
		return
	}
	data, serialErr := message.Serialize()
	if serialErr != nil {
		t.Fail()
		return
	}
	_, writeErr1 := dialConn.Write(data)
	if writeErr1 != nil {
		t.Fail()
		return
	}

	time.Sleep(1*time.Second)

	_, writeErr2 := dialConn.Write(data)
	if writeErr2 != nil {
		t.Fail()
		return
	}

	time.Sleep(1*time.Second)

	_, writeErr3 := dialConn.Write(data)
	if writeErr3 != nil {
		t.Fail()
		return
	}
}