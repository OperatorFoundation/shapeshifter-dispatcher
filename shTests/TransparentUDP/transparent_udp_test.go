package TransparentUDP

import (
	"net"
	"testing"
	"time"
)

func TestTransparentUDP(t *testing.T) {
	dialConn, dialError := net.Dial("udp", "127.0.0.1:1443")
	if dialError != nil {
		t.Fail()
		return
	}

	time.Sleep(1*time.Second)

	_, writeErr1 := dialConn.Write([]byte("data1"))
	if writeErr1 != nil {
		t.Fail()
		return
	}

	time.Sleep(1*time.Second)

	_, writeErr2 := dialConn.Write([]byte("data2"))
	if writeErr2 != nil {
		t.Fail()
		return
	}

	time.Sleep(1*time.Second)

	_, writeErr3 := dialConn.Write([]byte("data1"))
	if writeErr3 != nil {
		t.Fail()
		return
	}
}