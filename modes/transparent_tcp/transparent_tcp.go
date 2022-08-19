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
	commonLog "github.com/OperatorFoundation/shapeshifter-dispatcher/common/log"
	"github.com/OperatorFoundation/shapeshifter-dispatcher/common/pt_extras"
	"github.com/OperatorFoundation/shapeshifter-dispatcher/modes"
	pt "github.com/OperatorFoundation/shapeshifter-ipc/v3"
	"github.com/kataras/golog"
	"golang.org/x/net/proxy"
	"net"
	"net/url"
)

func ClientSetup(socksAddr string, ptClientProxy *url.URL, names []string, options string, enableLocket bool, stateDir string) (launched bool) {
	return modes.ClientSetupTCP(socksAddr, ptClientProxy, names, options, clientHandler, enableLocket, stateDir)
}

func clientHandler(name string, options string, conn net.Conn, proxyURI *url.URL) {
	var dialer proxy.Dialer
	dialer = proxy.Direct
	if proxyURI != nil {
		var err error
		dialer, err = proxy.FromURL(proxyURI, proxy.Direct)
		if err != nil {
			// This should basically never happen, since config protocol
			// verifies this.
			fmt.Println("-> failed to obtain dialer", proxyURI, proxy.Direct)
			golog.Errorf("(%s) - failed to obtain proxy dialer: %s", commonLog.ElideError(err))
			conn.Close()
			return
		}
	}

	// Deal with arguments.
	transport, argsToDialerErr := pt_extras.ArgsToDialer(name, options, dialer)
	if argsToDialerErr != nil {
		golog.Errorf("Error creating a transport with the provided options: %v", options)
		golog.Errorf("Error: %v", argsToDialerErr.Error())
		println("-> Error creating a transport with the provided options: ", options)
		println("-> Error: ", argsToDialerErr.Error())
		conn.Close()

		return
	}

	if conn == nil {
		println("--> Application connection is nil")
		golog.Errorf("%s - closed connection. Application connection is nil", name)
	}

	fmt.Println("Dialing ")
	remote, dialErr := transport.Dial()
	if dialErr != nil {

		println("--> Unable to dial transport server: ", dialErr.Error())
		println("-> Name: ", name)
		println("-> Options: ", options)
		golog.Errorf("--> Unable to dial transport server: %v", dialErr.Error())
		conn.Close()
		return
	}

	if remote == nil {
		println("--> Transport server connection is nil.")
		golog.Errorf("%s - closed connection. Transport server connection is nil", name)
		conn.Close()
	}

	if err := modes.CopyLoop(conn, remote); err != nil {
		golog.Warnf("%s(%s) - closed connection: %s", name, commonLog.ElideError(err))
		println("%s(%s) - closed connection: %s", name, commonLog.ElideError(err))
	} else {
		golog.Infof("%s(%s) - closed connection", name)
		println("%s(%s) - closed connection", name)
	}
}

func ServerSetup(ptServerInfo pt.ServerInfo, statedir string, options string, enableLocket bool) (launched bool) {
	return modes.ServerSetupTCP(ptServerInfo, statedir, options, serverHandler, enableLocket)
}

func serverHandler(name string, remote net.Conn, info *pt.ServerInfo) {
	// Connect to the orport.
	orConn, err := pt.DialOr(info, remote.RemoteAddr().String(), name)
	if err != nil {
		print("failed to connect to ORPort: ")
		println(commonLog.ElideError(err))
		golog.Errorf("%s - failed to connect to ORPort: %s", name, commonLog.ElideError(err))
		remote.Close()
		return
	}

	if err = modes.CopyLoop(orConn, remote); err != nil {
		print("closed a connection: ")
		println(commonLog.ElideError(err))
		golog.Warnf("%s - closed connection: %s", name, commonLog.ElideError(err))
	} else {
		golog.Infof("%s - closed connection", name)
	}
}
