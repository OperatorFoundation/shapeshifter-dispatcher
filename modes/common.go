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
	"github.com/OperatorFoundation/shapeshifter-dispatcher/common/log"
	"github.com/OperatorFoundation/shapeshifter-dispatcher/common/pt_extras"
	pt "github.com/OperatorFoundation/shapeshifter-ipc/v2"
	"golang.org/x/net/proxy"
	"net"
	"net/url"
)

type ConnState struct {
	Conn    net.Conn
	Waiting bool
}

type ConnTracker map[string]ConnState

type ClientHandlerTCP func(target string, name string, options string, conn net.Conn, proxyURI *url.URL)

type ClientHandlerUDP func(target string, name string, options string, conn *net.UDPConn, proxyURI *url.URL)
type ServerHandler func(name string, remote net.Conn, info *pt.ServerInfo)

func NewConnState() ConnState {
	return ConnState{nil, true}
}

func OpenConnection(tracker *ConnTracker, addr string, target string, name string, options string, proxyURI *url.URL) {
	newConn := NewConnState()
	(*tracker)[addr] = newConn

	go dialConn(tracker, addr, target, name, options, proxyURI)
}

func dialConn(tracker *ConnTracker, addr string, target string, name string, options string, proxyURI *url.URL) {
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
			log.Errorf("(%s) - failed to obtain proxy dialer: %s", target, log.ElideError(err))
			return
		}

	}

	fmt.Println("Dialing....")

	// Deal with arguments.
	transport, argsToDialerErr := pt_extras.ArgsToDialer(target, name, options, dialer)
	if argsToDialerErr != nil {
		log.Errorf("Error creating a transport with the provided options: %s", options)
		log.Errorf("Error: %s", argsToDialerErr)
		return
	}
	fmt.Println("Dialing ", target)
	remote, dialError := transport.Dial()
	if dialError != nil {
		fmt.Println("outgoing connection failed", dialError)
		log.Errorf("(%s) - outgoing connection failed: %s", target, log.ElideError(dialError))
		fmt.Println("Failed")
		delete(*tracker, addr)
		return
	}

	fmt.Println("Success")

	(*tracker)[addr] = ConnState{remote, false}
}

func ServerAcceptLoop(name string, ln net.Listener, info *pt.ServerInfo, serverHandler ServerHandler) {
	for {
		conn, err := ln.Accept()
		fmt.Println("accepted")
		if err != nil {
			if e, ok := err.(net.Error); ok && !e.Temporary() {
				log.Errorf("ServerAcceptLoop failed")
				_ = ln.Close()
				return
			}
			continue
		}

		go serverHandler(name, conn, info)
	}
}
