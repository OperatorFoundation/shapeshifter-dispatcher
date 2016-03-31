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
package proxy_transparent

import (
	"fmt"
	"io"
	golog "log"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/net/proxy"

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

func ClientSetup(termMon *termmon.TermMonitor, target string) (launched bool, listeners []net.Listener) {
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

		fmt.Println("Listening ", socksAddr)
		ln, err := net.Listen("tcp", socksAddr)
		if err != nil {
			log.Errorf("failed to listen %s %s", name, err.Error())
			continue
		}

		go clientAcceptLoop(target, termMon, f, ln, ptClientProxy)

		log.Infof("%s - registered listener: %s", name, ln.Addr())

		listeners = append(listeners, ln)
		launched = true
	}

	return
}

func clientAcceptLoop(target string, termMon *termmon.TermMonitor, f base.ClientFactory, ln net.Listener, proxyURI *url.URL) error {
	defer ln.Close()
	for {
		conn, err := ln.Accept()
		fmt.Println("Accepted")
		if err != nil {
			if e, ok := err.(net.Error); ok && !e.Temporary() {
				return err
			}
			continue
		}
		go clientHandler(target, termMon, f, conn, proxyURI)
	}
}

func clientHandler(target string, termMon *termmon.TermMonitor, f base.ClientFactory, conn net.Conn, proxyURI *url.URL) {
	defer conn.Close()
	termMon.OnHandlerStart()
	defer termMon.OnHandlerFinish()

	fmt.Println("handling...")

	name := f.Transport().Name()

	fmt.Println("Transport is", name)

	// Deal with arguments.
	args, err := f.ParseArgs(&pt.Args{})
	if err != nil {
		fmt.Println("Invalid arguments")
		log.Errorf("%s(%s) - invalid arguments: %s", name, target, err)
		return
	}

	fmt.Println("Making dialer...")

	// Obtain the proxy dialer if any, and create the outgoing TCP connection.
	dialFn := proxy.Direct.Dial
	if proxyURI != nil {
		dialer, err := proxy.FromURL(proxyURI, proxy.Direct)
		if err != nil {
			// This should basically never happen, since config protocol
			// verifies this.
			fmt.Println("failed to obtain dialer", proxyURI, proxy.Direct)
			log.Errorf("%s(%s) - failed to obtain proxy dialer: %s", name, target, log.ElideError(err))
			return
		}
		dialFn = dialer.Dial
	}

	fmt.Println("Dialing...")

	remote, err := f.Dial("tcp", target, dialFn, args)
	if err != nil {
		fmt.Println("outgoing connection failed")
		log.Errorf("%s(%s) - outgoing connection failed: %s", name, target, log.ElideError(err))
		return
	}
	defer remote.Close()

	fmt.Println("copying...")

	if err = copyLoop(conn, remote); err != nil {
		log.Warnf("%s(%s) - closed connection: %s", name, target, log.ElideError(err))
	} else {
		log.Infof("%s(%s) - closed connection", name, target)
	}

	fmt.Println("done")

	return
}

func ServerSetup(termMon *termmon.TermMonitor, bindaddrString string) (launched bool, listeners []net.Listener) {
	ptServerInfo, err := pt.ServerSetup(transports.Transports())
	if err != nil {
		golog.Fatal(err)
	}

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

		go serverAcceptLoop(termMon, f, ln, &ptServerInfo)

		log.Infof("%s - registered listener: %s", name, log.ElideAddr(ln.Addr().String()))

		listeners = append(listeners, ln)
		launched = true
	}

	return
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

func serverAcceptLoop(termMon *termmon.TermMonitor, f base.ServerFactory, ln net.Listener, info *pt.ServerInfo) error {
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
		go serverHandler(termMon, f, conn, info)
	}
}

func serverHandler(termMon *termmon.TermMonitor, f base.ServerFactory, conn net.Conn, info *pt.ServerInfo) {
	defer conn.Close()
	termMon.OnHandlerStart()
	defer termMon.OnHandlerFinish()

	name := f.Transport().Name()
	addrStr := log.ElideAddr(conn.RemoteAddr().String())
	fmt.Println("handling", name)
	log.Infof("%s(%s) - new connection", name, addrStr)

	// Instantiate the server transport method and handshake.
	remote, err := f.WrapConn(conn)
	if err != nil {
		fmt.Println("handshake failed")
		log.Warnf("%s(%s) - handshake failed: %s", name, addrStr, log.ElideError(err))
		return
	}

	// Connect to the orport.
	orConn, err := pt.DialOr(info, conn.RemoteAddr().String(), name)
	if err != nil {
		fmt.Println("OR conn failed", info, conn.RemoteAddr(), name)
		log.Errorf("%s(%s) - failed to connect to ORPort: %s", name, addrStr, log.ElideError(err))
		return
	}
	defer orConn.Close()

	if err = copyLoop(orConn, remote); err != nil {
		log.Warnf("%s(%s) - closed connection: %s", name, addrStr, log.ElideError(err))
	} else {
		log.Infof("%s(%s) - closed connection", name, addrStr)
	}

	return
}

func copyLoop(a net.Conn, b net.Conn) error {
	// Note: b is always the pt connection.  a is the SOCKS/ORPort connection.
	errChan := make(chan error, 2)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		defer b.Close()
		defer a.Close()
		_, err := io.Copy(b, a)
		errChan <- err
	}()
	go func() {
		defer wg.Done()
		defer a.Close()
		defer b.Close()
		_, err := io.Copy(a, b)
		errChan <- err
	}()

	// Wait for both upstream and downstream to close.  Since one side
	// terminating closes the other, the second error in the channel will be
	// something like EINVAL (though io.Copy() will swallow EOF), so only the
	// first error is returned.
	wg.Wait()
	if len(errChan) > 0 {
		return <-errChan
	}

	return nil
}
