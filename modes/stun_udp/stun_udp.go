/*
 * Copyright (c) 2014-2015, Yawning Angel <yawning at torproject dot org>
 * All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are met:
 *
 *  * Redistributions of source code must retain the above copyright notice,
 *    this list of conditions and the following disclaimer.
 *
 *  * Redistributions in binary form must reproduce the above copyright notice,
 *    this list of conditions and the following disclaimer in the documentation
 *    and/or other materials provided with the distribution.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
 * AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
 * IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
 * ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE
 * LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
 * CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
 * SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
 * INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
 * CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
 * ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
 * POSSIBILITY OF SUCH DAMAGE.
 */

// Go language Tor Pluggable Transport suite.  Works only as a managed
// client/server.
package stun_udp

import (
	"io"
	"fmt"
	golog "log"
	"net"
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/net/proxy"

	"github.com/willscott/goturn"

	"git.torproject.org/pluggable-transports/goptlib.git"
	"github.com/OperatorFoundation/obfs4/common/log"
	"github.com/OperatorFoundation/obfs4/common/termmon"
	"github.com/OperatorFoundation/obfs4/transports"
	"github.com/OperatorFoundation/obfs4/transports/base"
)

const (
	obfs4proxyVersion = "0.0.7-dev"
	obfs4proxyLogFile = "obfs4proxy.log"
	socksAddr         = "127.0.0.1:1234"
)

var stateDir string

type ConnState struct {
	Conn *net.Conn
	Waiting bool
}

func NewConnState() ConnState {
	return ConnState{nil, true}
}

type ConnTracker map[string]ConnState

func ClientSetup(termMon *termmon.TermMonitor, target string) bool {
	methodNames := [...]string{"obfs2"}
	var ptClientProxy *url.URL = nil

	// Launch each of the client listeners.
	for _, name := range methodNames {
		t := transports.Get(name)
		if t == nil {
			log.Errorf("no such transport is supported: %s", name)
			continue
		}

		f, err := t.ClientFactory(stateDir)
		if err != nil {
			log.Errorf("failed to get ClientFactory: %s", name)
			continue
		}

		udpAddr, err := net.ResolveUDPAddr("udp", socksAddr)
		if err != nil {
			fmt.Println("Error resolving address", socksAddr)
		}

		fmt.Println("@@@ Listening ", name, socksAddr)
		ln, err := net.ListenUDP("udp", udpAddr)
		if err != nil {
			log.Errorf("failed to listen %s %s", name, err.Error())
			continue
		}

		go clientHandler(target, termMon, f, ln, ptClientProxy)

		log.Infof("%s - registered listener: %s", name, ln)
	}

	return true
}

func clientHandler(target string, termMon *termmon.TermMonitor, f base.ClientFactory, conn *net.UDPConn, proxyURI *url.URL) {
	defer conn.Close()
	termMon.OnHandlerStart()
	defer termMon.OnHandlerFinish()

	fmt.Println("@@@ handling...")

  tracker := make(ConnTracker)

	name := f.Transport().Name()

	fmt.Println("Transport is", name)

  buf := make([]byte, 1024)

  // Receive UDP packets and forward them over transport connections forever
	for {
		n, addr, err := conn.ReadFromUDP(buf)
		fmt.Println("Received ",string(buf[0:n]), " from ",addr)

		if err != nil {
      fmt.Println("Error: ",err)
		}

    fmt.Println(tracker)

		if state, ok := tracker[addr.String()]; ok {
			// There is an open transport connection, or a connection attempt is in progress.

			if state.Waiting {
				// The connection attempt is in progress.
				// Drop the packet.
				fmt.Println("recv: waiting")
			} else {
				// There is an open transport connection.
				// Send the packet through the transport.
				fmt.Println("recv: write")
				fmt.Println("writing...")
				(*state.Conn).Write(buf)
			}
    } else {
			// There is not an open transport connection and a connection attempt is not in progress.
			// Open a transport connection.

      fmt.Println("Opening connection to ", target)

			openConnection(&tracker, addr.String(), target, termMon, f, proxyURI)

			// Drop the packet.
			fmt.Println("recv: Open")
		}
	}
}

func openConnection(tracker *ConnTracker, addr string, target string, termMon *termmon.TermMonitor, f base.ClientFactory, proxyURI *url.URL) {
	fmt.Println("Making dialer...")

	newConn := NewConnState()
	(*tracker)[addr]=newConn

	go dialConn(tracker, addr, target, f, proxyURI)
}

func dialConn(tracker *ConnTracker, addr string, target string, f base.ClientFactory, proxyURI *url.URL) {
	// Obtain the proxy dialer if any, and create the outgoing TCP connection.
	dialFn := proxy.Direct.Dial
	if proxyURI != nil {
		dialer, err := proxy.FromURL(proxyURI, proxy.Direct)
		if err != nil {
			// This should basically never happen, since config protocol
			// verifies this.
			fmt.Println("failed to obtain dialer", proxyURI, proxy.Direct)
			log.Errorf("(%s) - failed to obtain proxy dialer: %s", target, log.ElideError(err))
			return
		}
		dialFn = dialer.Dial
	}

	fmt.Println("Dialing....")

	// Deal with arguments.
	args, err := f.ParseArgs(&pt.Args{})
	if err != nil {
		fmt.Println("Invalid arguments")
		log.Errorf("(%s) - invalid arguments: %s", target, err)
		delete(*tracker, addr)
		return
	}

	fmt.Println("Dialing ", target)
	remote, err := f.Dial("tcp", target, dialFn, args)
	if err != nil {
		fmt.Println("outgoing connection failed", err)
		log.Errorf("(%s) - outgoing connection failed: %s", target, log.ElideError(err))
		fmt.Println("Failed")
		delete(*tracker, addr)
		return
	}

	fmt.Println("Success")

	(*tracker)[addr]=ConnState{&remote, false}
}

func ServerSetup(termMon *termmon.TermMonitor, bindaddrString string, target string) bool {
	fmt.Println("ServerSetup")

	bindaddrs, _ := getServerBindaddrs(bindaddrString)

	for _, bindaddr := range bindaddrs {
		name := bindaddr.MethodName
		fmt.Println("bindaddr", bindaddr)
		t := transports.Get(name)
		if t == nil {
			fmt.Println(name, "no such transport is supported")
			continue
		}

		f, err := t.ServerFactory(stateDir, &bindaddr.Options)
		if err != nil {
			fmt.Println(name, err.Error())
			continue
		}

		ln, err := net.ListenTCP("tcp", bindaddr.Addr)
		if err != nil {
			fmt.Println(name, err.Error())
			continue
		}

		go serverAcceptLoop(termMon, f, ln, target)

		log.Infof("%s - registered listener: %s", name, log.ElideAddr(ln.Addr().String()))
	}

	return true
}

func getServerBindaddrs(serverBindaddr string) ([]pt.Bindaddr, error) {
	var result []pt.Bindaddr

	for _, spec := range strings.Split(serverBindaddr, ",") {
		var bindaddr pt.Bindaddr

		parts := strings.SplitN(spec, "-", 2)
		if len(parts) != 2 {
			fmt.Println("TOR_PT_SERVER_BINDADDR: doesn't contain \"-\"", spec)
			return nil, nil
		}
		bindaddr.MethodName = parts[0]
		addr, err := resolveAddr(parts[1])
		if err != nil {
			fmt.Println("TOR_PT_SERVER_BINDADDR: ", spec, err.Error())
			return nil, nil
		}
		bindaddr.Addr = addr
		//		bindaddr.Options = optionsMap[bindaddr.MethodName]
		result = append(result, bindaddr)
	}

	return result, nil
}

// Resolve an address string into a net.TCPAddr. We are a bit more strict than
// net.ResolveTCPAddr; we don't allow an empty host or port, and the host part
// must be a literal IP address.
func resolveAddr(addrStr string) (*net.TCPAddr, error) {
	ipStr, portStr, err := net.SplitHostPort(addrStr)
	if err != nil {
		// Before the fixing of bug #7011, tor doesn't put brackets around IPv6
		// addresses. Split after the last colon, assuming it is a port
		// separator, and try adding the brackets.
		parts := strings.Split(addrStr, ":")
		if len(parts) <= 2 {
			return nil, err
		}
		addrStr := "[" + strings.Join(parts[:len(parts)-1], ":") + "]:" + parts[len(parts)-1]
		ipStr, portStr, err = net.SplitHostPort(addrStr)
	}
	if err != nil {
		return nil, err
	}
	if ipStr == "" {
		return nil, net.InvalidAddrError(fmt.Sprintf("address string %q lacks a host part", addrStr))
	}
	if portStr == "" {
		return nil, net.InvalidAddrError(fmt.Sprintf("address string %q lacks a port part", addrStr))
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, net.InvalidAddrError(fmt.Sprintf("not an IP string: %q", ipStr))
	}
	port, err := parsePort(portStr)
	if err != nil {
		return nil, err
	}
	return &net.TCPAddr{IP: ip, Port: port}, nil
}

func parsePort(portStr string) (int, error) {
	port, err := strconv.ParseUint(portStr, 10, 16)
	return int(port), err
}

func serverAcceptLoop(termMon *termmon.TermMonitor, f base.ServerFactory, ln net.Listener, target string) error {
	defer ln.Close()
	for {
		conn, err := ln.Accept()
		fmt.Println("accepted")
		if err != nil {
			if e, ok := err.(net.Error); ok && !e.Temporary() {
				return err
			}
			continue
		}
		go serverHandler(termMon, f, conn, target)
	}
}

func serverHandler(termMon *termmon.TermMonitor, f base.ServerFactory, conn net.Conn, target string) {
	var header *turn.StunHeader

	defer conn.Close()
	termMon.OnHandlerStart()
	defer termMon.OnHandlerFinish()

	name := f.Transport().Name()
	addrStr := log.ElideAddr(conn.RemoteAddr().String())
	fmt.Println("### handling", name)
	log.Infof("%s(%s) - new connection", name, addrStr)

	// Instantiate the server transport method and handshake.
	remote, err := f.WrapConn(conn)
	if err != nil {
		fmt.Println("handshake failed", err)
		log.Warnf("%s(%s) - handshake failed: %s", name, addrStr, log.ElideError(err))
		return
	}

	serverAddr, err := net.ResolveUDPAddr("udp",target)
	if err != nil {
		golog.Fatal(err)
	}

	localAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		golog.Fatal(err)
	}

	dest, err := net.DialUDP("udp", localAddr, serverAddr)
	if err != nil {
		golog.Fatal(err)
	}

  fmt.Println("pumping")

	defer dest.Close()

	headerBuffer := make([]byte, 20)

	for {
		fmt.Println("reading...")
		// Read the incoming connection into the buffer.
	  _, err := io.ReadFull(remote, headerBuffer)
		if err != nil {
			fmt.Println("read error")
			break
		}

		header=&turn.StunHeader{}
		header.Decode(headerBuffer)

		fmt.Println(header.Length)

		fmt.Println("reading data")

		readBuffer := make([]byte, header.Length)
		_, err = io.ReadFull(remote, readBuffer)
		if err != nil {
			fmt.Println("read error")
			break
		}

    writeBuffer := append(headerBuffer, readBuffer...)

		dest.Write(writeBuffer)
  }
}
