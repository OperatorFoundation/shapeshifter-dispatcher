package main

import (
	socks "github.com/OperatorFoundation/shapeshifter-dispatcher/common/socks5"
	"net"
	"testing"
	"time"
)

func TestSocksTCP(t *testing.T) {
	dialConn, dialError := net.Dial("tcp", "127.0.0.1:1443")
	if dialError != nil {
		t.Fail()
		return
	}

	time.Sleep(1 * time.Second)
	c := new(socks.TestReadWriter)
	req := c.ToRequest()
	var err error
	var method byte

	// VER = 05, NMETHODS = 01, METHODS = [09]
	//Method 9 is the json parameter block authentication
	_, hexErr := c.WriteHex("050109")
	if hexErr != nil {
		t.Error("negotiateAuth(jsonParameterBlock) could not be decoded")
	}
	if method, err = req.NegotiateAuth(false); err != nil {
		t.Error("negotiateAuth(jsonParameterBlock) failed:", err)
	}
	if method != socks.AuthJsonParameterBlock {
		t.Error("negotiateAuth(jsonParameterBlock) unexpected method:", method)
	}
	if msg := c.ReadHex(); msg != "0509" {
		t.Error("negotiateAuth(jsonParameterBlock) invalid response:", msg)
	}
	_, writeErr := dialConn.Write([]byte("data"))
	if writeErr != nil {
		t.Fail()
		return
	}

}
