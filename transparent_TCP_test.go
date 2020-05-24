package main

import (
	"net"
	"testing"
	"time"
)

func TestTransparentTCP(t *testing.T) {
	dialConn, dialError := net.Dial("tcp", "127.0.0.1:1443")
	if dialError != nil {
		t.Fail()
		return
	}

	time.Sleep(1*time.Second)

	_, writeErr := dialConn.Write([]byte("data"))
	if writeErr != nil {
		t.Fail()
		return
	}

	time.Sleep(1*time.Second)

}