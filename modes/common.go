/*
MIT License

Copyright (c) 2020 Operator Foundation

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NON-INFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package modes

import (
	"fmt"
	"net"
	"net/url"

	locketgo "github.com/OperatorFoundation/locket-go"
	"github.com/OperatorFoundation/shapeshifter-dispatcher/common/log"
	"github.com/OperatorFoundation/shapeshifter-dispatcher/common/pt_extras"
	pt "github.com/OperatorFoundation/shapeshifter-ipc/v3"
	"github.com/kataras/golog"
	"golang.org/x/net/proxy"
)

type ConnState struct {
	Conn    net.Conn
	Waiting bool
}

type ConnTracker map[string]ConnState

type ClientHandlerTCP func(name string, options string, conn net.Conn, proxyURI *url.URL)

type ClientHandlerUDP func(name string, options string, conn *net.UDPConn, proxyURI *url.URL)

type ServerHandler func(name string, remote net.Conn, info *pt.ServerInfo)

func NewConnState() ConnState {
	return ConnState{nil, true}
}

func OpenConnection(tracker *ConnTracker, addr string, name string, options string, proxyURI *url.URL) {
	newConn := NewConnState()
	(*tracker)[addr] = newConn

	go dialConn(tracker, addr, name, options, proxyURI)
}

func dialConn(tracker *ConnTracker, addr string, name string, options string, proxyURI *url.URL) {
	// Obtain the proxy dialer if any, and create the outgoing TCP connection.
	var dialer proxy.Dialer
	dialer = proxy.Direct
	if proxyURI != nil {
		var err error
		dialer, err = proxy.FromURL(proxyURI, proxy.Direct)
		if err != nil {
			// This should basically never happen, since config protocol
			// verifies this.
			fmt.Println("failed to obtain dialer", proxyURI, proxy.Direct)
			golog.Error("(%s) - failed to obtain proxy dialer")
			return
		}

	}

	println("Dialing....")

	// Deal with arguments.
	transport, argsToDialerErr := pt_extras.ArgsToDialer(name, options, dialer)

	if argsToDialerErr != nil {
		log.Errorf("Error creating a transport with the provided options: %s", options)
		log.Errorf("Error: %s", argsToDialerErr)
		return
	}
	fmt.Println("Dialing ")
	remote, dialError := transport.Dial()
	if dialError != nil {
		fmt.Println("outgoing connection failed: ", dialError)
		golog.Error("(%s) - outgoing connection failed")
		println("Failed")
		delete(*tracker, addr)
		return
	}

	println("Success")

	(*tracker)[addr] = ConnState{remote, false}
}

func ServerAcceptLoop(name string, ln net.Listener, info *pt.ServerInfo, serverHandler ServerHandler, enableLocket bool, stateDir string) {
	for {
		conn, err := ln.Accept()
		fmt.Println("accepted")
		if err != nil {
			print("Received an error while attempting to accept a connection:")
			print(err.Error())

			if e, ok := err.(net.Error); ok && !e.Temporary() {
				log.Errorf("ServerAcceptLoop failed")
				_ = ln.Close()
				return
			}
			continue
		}

		if enableLocket {
			locketConn, locketError := locketgo.NewLocketConn(conn, stateDir, "DispatcherServer")
			if locketError != nil {
				golog.Error("server failed to enable Locket")
				conn.Close()
				return
			}

			conn = locketConn
		}

		go serverHandler(name, conn, info)
	}
}
