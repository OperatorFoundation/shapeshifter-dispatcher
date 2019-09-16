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
package transparent_tcp

import (
	"fmt"
	options2 "github.com/OperatorFoundation/shapeshifter-dispatcher/common"
	"github.com/OperatorFoundation/shapeshifter-dispatcher/common/pt_extras"
	"github.com/OperatorFoundation/shapeshifter-transports/transports/Dust"
	replicant "github.com/OperatorFoundation/shapeshifter-transports/transports/Replicant"
	"github.com/OperatorFoundation/shapeshifter-transports/transports/meeklite"
	"github.com/OperatorFoundation/shapeshifter-transports/transports/obfs2"
	"io"
	"net"
	"net/url"
	"strings"
	"sync"

	"github.com/OperatorFoundation/shapeshifter-dispatcher/common/log"
	"github.com/OperatorFoundation/shapeshifter-dispatcher/common/termmon"
	"github.com/OperatorFoundation/shapeshifter-ipc"
	//"github.com/OperatorFoundation/shapeshifter-transports/transports/obfs2"
	"github.com/OperatorFoundation/shapeshifter-transports/transports/obfs4"
	"github.com/OperatorFoundation/shapeshifter-transports/transports/shadow"
)

var stateDir string

func ClientSetup(termMon *termmon.TermMonitor, socksAddr string, target string, ptClientProxy *url.URL, names []string, options string) (launched bool, listeners []net.Listener) {
	// Launch each of the client listeners.
	for _, name := range names {
		ln, err := net.Listen("tcp", socksAddr)
		if err != nil {
			log.Errorf("failed to listen %s %s", name, err.Error())
			continue
		}

		go clientAcceptLoop(target, termMon, name, options, ln, ptClientProxy)

		log.Infof("%s - registered listener: %s", name, ln.Addr())

		listeners = append(listeners, ln)
		launched = true
	}

	return
}

func clientAcceptLoop(target string, termMon *termmon.TermMonitor, name string, options string, ln net.Listener, proxyURI *url.URL) error {
	defer ln.Close()
	for {
		conn, err := ln.Accept()
		if err != nil {
			if e, ok := err.(net.Error); ok && !e.Temporary() {
				return err
			}
			continue
		}
		go clientHandler(target, termMon, name, options, conn, proxyURI)
	}
}

func clientHandler(target string, termMon *termmon.TermMonitor, name string, options string, conn net.Conn, proxyURI *url.URL) {
	defer conn.Close()
	termMon.OnHandlerStart()
	defer termMon.OnHandlerFinish()

	var dialer func() (net.Conn, error)
//this is where the refactoring begins
	args, argsErr := options2.ParseOptions(options)
	if argsErr != nil {
		log.Errorf("Error parsing transport options: %s", options)
		return
	}

	// Deal with arguments.

	transport, dialerErr := pt_extras.ArgsToDialer(target, name, args)
	if dialerErr != nil {
		log.Errorf("Error parsing transport-specific options: %s (%s)", args, dialerErr)
		return
	}
	dialer = transport.Dial
	f := dialer

	// Obtain the proxy dialer if any, and create the outgoing TCP connection.
	// dialFn := proxy.Direct.Dial
	// if proxyURI != nil {
	// 	dialer, err := proxy.FromURL(proxyURI, proxy.Direct)
	// 	if err != nil {
	// 		// This should basically never happen, since config protocol
	// 		// verifies this.
	// 		log.Errorf("%s(%s) - failed to obtain proxy dialer: %s", name, target, log.ElideError(err))
	// 		return
	// 	}
	// 	dialFn = dialer.Dial
	// }

	// FIXME - use dialFn if a proxy is needed to connect to the network
	remote, err := f()
	// if err != nil {
	// 	log.Errorf("%s(%s) - outgoing connection failed: %s", name, target, log.ElideError(err))
	// 	return
	// }
	if err != nil {
		fmt.Errorf("outgoing connection failed %q %q", f, target)
		return
	}

	defer remote.Close()

	if err := copyLoop(conn, remote); err != nil {
		log.Warnf("%s(%s) - closed connection: %s", name, target, log.ElideError(err))
	} else {
		log.Infof("%s(%s) - closed connection", name, target)
	}

	return
}



func ServerSetup(termMon *termmon.TermMonitor, bindaddrString string, ptServerInfo pt.ServerInfo, statedir string, options string) (launched bool, listeners []net.Listener) {
	// Launch each of the server listeners.
	for _, bindaddr := range ptServerInfo.Bindaddrs {
		name := bindaddr.MethodName

		var listen func(address string) net.Listener

		args, argsErr := pt.ParsePT2ServerParameters(options)
		if argsErr != nil {
			log.Errorf("Error parsing transport options: %s", options)
			return
		}

		// Deal with arguments.
		switch name {
		case "obfs2":
			transport := obfs2.NewObfs2Transport()
			listen = transport.Listen
		case "obfs4":
			transport := obfs4.NewObfs4Server(statedir)
			listen = transport.Listen
		case "Replicant":
			shargs, aok := args["Replicant"]
			if !aok {
				return false, nil
			}

			_, ok := shargs.Get("config")
			if !ok {
				return false, nil
			}
			transport := replicant.New(replicant.Config{})
			listen = transport.Listen
		case "Dust":
			shargs, aok := args["Dust"]
			if !aok {
				return false, nil
			}

			idPath, ok := shargs.Get("Url")
			if !ok {
				return false, nil
			}
			transport := Dust.NewDustServer(idPath)
			listen = transport.Listen
		case "meeklite":
			shargs, aok := args["meeklite"]
			if !aok {
				return false, nil
			}

			Url, ok := shargs.Get("Url")
			if !ok {
				return false, nil
			}

			Front, ok2 := shargs.Get("Front")
			if !ok2 {
				return false, nil
			}
			transport := meeklite.NewMeekTransportWithFront(Url, Front)
			listen = transport.Listen
		case "shadow":
			shargs, aok := args["shadow"]
			if !aok {
				return false, nil
			}

			password, ok := shargs.Get("password")
			if !ok {
				return false, nil
			}

			cipherName, ok2 := shargs.Get("cipherName")
			if !ok2 {
				return false, nil
			}

			transport := shadow.NewShadowServer(password, cipherName)
			listen = transport.Listen
		default:
			log.Errorf("Unknown transport: %s", name)
			return false, nil
		}

		f := listen

		transportLn := f(bindaddr.Addr.String())

		go serverAcceptLoop(termMon, name, transportLn, &ptServerInfo)

		log.Infof("%s - registered listener: %s", name, log.ElideAddr(bindaddr.Addr.String()))

		listeners = append(listeners, transportLn)
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
			fmt.Errorf("TOR_PT_SERVER_BINDADDR: doesn't contain \"-\" %q", spec)
			return nil, nil
		}
		bindaddr.MethodName = parts[0]
		addr, err := pt.ResolveAddr(parts[1])
		if err != nil {
			fmt.Errorf("TOR_PT_SERVER_BINDADDR: %q %q", spec, err.Error())
			return nil, nil
		}
		bindaddr.Addr = addr
		//		bindaddr.Options = optionsMap[bindaddr.MethodName]
		result = append(result, bindaddr)
	}

	return result, nil
}

func serverAcceptLoop(termMon *termmon.TermMonitor, name string, ln net.Listener, info *pt.ServerInfo) error {
	defer ln.Close()
	for {
		conn, err := ln.Accept()
		if err != nil {
			if e, ok := err.(net.Error); ok && !e.Temporary() {
				return err
			}
			continue
		}
		go serverHandler(termMon, name, conn, info)
	}
}

func serverHandler(termMon *termmon.TermMonitor, name string, remote net.Conn, info *pt.ServerInfo) {
	termMon.OnHandlerStart()
	defer termMon.OnHandlerFinish()

	// Connect to the orport.
	orConn, err := pt.DialOr(info, remote.RemoteAddr().String(), name)
	if err != nil {
		log.Errorf("%s - failed to connect to ORPort: %s", name, log.ElideError(err))
		return
	}
	defer orConn.Close()

	if err = copyLoop(orConn, remote); err != nil {
		log.Warnf("%s - closed connection: %s", name, log.ElideError(err))
	} else {
		log.Infof("%s - closed connection", name)
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

