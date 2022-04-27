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
package pt_socks5

import (
	commonLog "github.com/OperatorFoundation/shapeshifter-dispatcher/common/log"
	"github.com/OperatorFoundation/shapeshifter-dispatcher/common/pt_extras"
	"github.com/OperatorFoundation/shapeshifter-dispatcher/common/socks5"
	"github.com/OperatorFoundation/shapeshifter-dispatcher/modes"
	pt "github.com/OperatorFoundation/shapeshifter-ipc/v3"
	"github.com/kataras/golog"
	"golang.org/x/net/proxy"
	"net"
	"net/url"
)

func ClientSetup(socksAddr string, ptClientProxy *url.URL, names []string, options string) (launched bool) {
	// Launch each of the client listeners.
	for _, name := range names {
		ln, err := net.Listen("tcp", socksAddr)
		if err != nil {
			_ = pt.CmethodError(name, err.Error())
			continue
		}

		go clientAcceptLoop(name, ln, ptClientProxy, options)
		pt.Cmethod(name, socks5.Version(), ln.Addr())

		golog.Infof("%s - registered listener: %s", name, ln.Addr())

		launched = true
	}
	pt.CmethodsDone()

	return
}

func clientAcceptLoop(name string, ln net.Listener, proxyURI *url.URL, options string) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			if e, ok := err.(net.Error); ok && !e.Temporary() {
				golog.Errorf("serverAcceptLoop failed")
				_ = ln.Close()
				return
			}
			continue
		}
		go clientHandler(name, conn, proxyURI, options)
	}
}

func clientHandler(name string, conn net.Conn, proxyURI *url.URL, options string) {
	var needOptions = options == ""

	// Read the client's SOCKS handshake.
	socksReq, err := socks5.Handshake(conn, needOptions)
	if err != nil {
		golog.Errorf("%s - client failed socks handshake: %s", name, err)
		conn.Close()
		return
	}
	addrStr := commonLog.ElideAddr(socksReq.Target)

	var dialer proxy.Dialer = proxy.Direct

	// Deal with arguments.

	transport, argsToDialerErr := pt_extras.ArgsToDialer(name, options, dialer)
	if argsToDialerErr != nil {
		golog.Errorf("Error creating a transport with the provided options: %s", options)
		golog.Errorf("Error: %s", argsToDialerErr)
		conn.Close()

		return
	}
	// Obtain the proxy dialer if any, and create the outgoing TCP connection.
	if proxyURI != nil {
		var proxyErr error
		dialer, proxyErr = proxy.FromURL(proxyURI, proxy.Direct)
		if proxyErr != nil {
			// This should basically never happen, since config protocol
			// verifies this.
			golog.Errorf("%s(%s) - failed to obtain proxy dialer: %s", name, addrStr, commonLog.ElideError(err))
			_ = socksReq.Reply(socks5.ReplyGeneralFailure)
			conn.Close()
			return
		}
	}

	remote, err2 := transport.Dial()
	if err2 != nil {
		golog.Errorf("%s(%s) - outgoing connection failed: %s", name, addrStr, commonLog.ElideError(err2))
		_ = socksReq.Reply(socks5.ErrorToReplyCode(err2))
		conn.Close()
		return
	}
	err = socksReq.Reply(socks5.ReplySucceeded)
	if err != nil {
		golog.Errorf("%s(%s) - SOCKS reply failed: %s", name, addrStr, commonLog.ElideError(err))
		conn.Close()
		return
	}

	if err = modes.CopyLoop(conn, remote); err != nil {
		golog.Warnf("%s(%s) - closed connection: %s", name, addrStr, commonLog.ElideError(err))
	} else {
		golog.Infof("%s(%s) - closed connection", name, addrStr)
	}

	return
}

func ServerSetup(ptServerInfo pt.ServerInfo, stateDir string, options string) (launched bool) {
	for _, bindaddr := range ptServerInfo.Bindaddrs {
		name := bindaddr.MethodName

		// Deal with arguments.
		listen, parseError := pt_extras.ArgsToListener(name, stateDir, options)
		if parseError != nil {
			return false
		}

		go func() {
			for {
				transportLn, LnError := listen(bindaddr.Addr.String())
				if LnError != nil {
					continue
				}
				golog.Infof("%s - registered listener: %s", name, commonLog.ElideAddr(bindaddr.Addr.String()))
				modes.ServerAcceptLoop(name, transportLn, &ptServerInfo, serverHandler)
				transportLnErr := transportLn.Close()
				if transportLnErr != nil {
					golog.Errorf("Listener close error: %s", transportLnErr.Error())
				}
			}
		}()

		launched = true
	}
	pt.SmethodsDone()

	return
}

func serverHandler(name string, remote net.Conn, info *pt.ServerInfo) {

	addrStr := commonLog.ElideAddr(remote.RemoteAddr().String())
	golog.Infof("%s(%s) - new connection", name, addrStr)

	// Connect to the orport.
	orConn, err := pt.DialOr(info, remote.RemoteAddr().String(), name)
	if err != nil {
		golog.Errorf("%s(%s) - failed to connect to ORPort: %s", name, addrStr, commonLog.ElideError(err))
		remote.Close()

		return
	}

	if err = modes.CopyLoop(orConn, remote); err != nil {
		golog.Warnf("%s(%s) - closed connection: %s", name, addrStr, commonLog.ElideError(err))
	} else {
		golog.Infof("%s(%s) - closed connection", name, addrStr)
	}

	return
}
