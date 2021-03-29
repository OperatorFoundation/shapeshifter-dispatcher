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
	"os"

	"github.com/OperatorFoundation/obfs4/common/log"
	commonLog "github.com/OperatorFoundation/shapeshifter-dispatcher/common/log"
	"github.com/OperatorFoundation/shapeshifter-dispatcher/common/pt_extras"
	"github.com/OperatorFoundation/shapeshifter-dispatcher/modes"
	pt "github.com/OperatorFoundation/shapeshifter-ipc/v2"

	"net"
	"net/url"

	"golang.org/x/net/proxy"
)

func ClientSetup(socksAddr string, target string, ptClientProxy *url.URL, names []string, options string) (launched bool) {
	return modes.ClientSetupTCP(socksAddr, target, ptClientProxy, names, options, clientHandler)
}

func clientHandler(target string, name string, options string, conn net.Conn, proxyURI *url.URL) {
	if conn == nil {
		fmt.Fprintln(os.Stderr, "--> Application connection is nil")
		log.Errorf("%s - closed connection. Application connection is nil", name)
		return
	}

	var dialer proxy.Dialer = proxy.Direct
	if proxyURI != nil {
		var err error
		dialer, err = proxy.FromURL(proxyURI, proxy.Direct)
		if err != nil {
			// This should basically never happen, since config protocol
			// verifies this.
			fmt.Println("-> failed to obtain dialer", proxyURI, proxy.Direct)
			log.Errorf("(%s) - failed to obtain proxy dialer: %s", target, commonLog.ElideError(err))
			conn.Close()
			return
		}
	}

	// Deal with arguments.
	transport, argsToDialerErr := pt_extras.ArgsToDialer(target, name, options, dialer)
	if argsToDialerErr != nil {
		log.Errorf("Error creating a transport with the provided options: %v", options)
		log.Errorf("Error: %v", argsToDialerErr.Error())
		fmt.Fprintln(os.Stderr, "-> Error creating a transport with the provided options: ", options)
		fmt.Fprintln(os.Stderr, "-> Error: ", argsToDialerErr.Error())
		conn.Close()
		return
	}

	fmt.Println("Dialing ", target)
	remote, dialErr := transport.Dial()
	if dialErr != nil {
		fmt.Fprintln(os.Stderr, "--> Unable to dial transport server: ", dialErr.Error())
		fmt.Fprintln(os.Stderr, "-> Name: ", name)
		fmt.Fprintln(os.Stderr, "-> Options: ", options)
		log.Errorf("--> Unable to dial transport server: %v", dialErr.Error())
		conn.Close()
		return
	}
	if remote == nil {
		fmt.Fprintln(os.Stderr, "--> Transport server connection is nil.")
		log.Errorf("%s - closed connection. Transport server connection is nil", name)
		conn.Close()
		return
	}

	if err := modes.CopyLoop(conn, remote); err != nil {
		log.Errorf("%s(%s) - closed connection: %s", name, target, commonLog.ElideError(err))
		fmt.Fprintf(os.Stderr, "%s(%s) - closed connection: %s", name, target, commonLog.ElideError(err))
	} else {
		log.Infof("%s(%s) - closed connection", name, target)
		fmt.Fprintf(os.Stderr, "%s(%s) - closed connection", name, target)
	}
}

func ServerSetup(ptServerInfo pt.ServerInfo, statedir string, options string) (launched bool) {
	return modes.ServerSetupTCP(ptServerInfo, statedir, options, serverHandler)
}

func serverHandler(name string, remote net.Conn, info *pt.ServerInfo) {
	// Connect to the orport.
	orConn, err := pt.DialOr(info, remote.RemoteAddr().String(), name)
	if err != nil {
		log.Errorf("%s - failed to connect to ORPort: %s", name, log.ElideError(err))
		remote.Close()
		return
	}

	if err = modes.CopyLoop(orConn, remote); err != nil {
		log.Warnf("%s - closed connection: %s", name, log.ElideError(err))
	} else {
		log.Infof("%s - closed connection", name)
	}
}
