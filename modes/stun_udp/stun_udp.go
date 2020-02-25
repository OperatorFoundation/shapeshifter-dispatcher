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
	"fmt"
	"github.com/OperatorFoundation/shapeshifter-dispatcher/modes"
	common "github.com/willscott/goturn/common"
	"io"
	golog "log"
	"net"
	"net/url"

	"github.com/willscott/goturn"

	"github.com/OperatorFoundation/shapeshifter-dispatcher/common/log"
	"github.com/OperatorFoundation/shapeshifter-ipc"
)

func ClientSetup(socksAddr string, target string, ptClientProxy *url.URL, names []string, options string) bool {
	return modes.ClientSetupUDP(socksAddr, target, ptClientProxy, names, options, clientHandler)
}

func clientHandler(target string, name string, options string, conn *net.UDPConn, proxyURI *url.URL) {

	//defers are never called due to infinite loop

	fmt.Println("@@@ handling...")

	tracker := make(modes.ConnTracker)

	fmt.Println("Transport is", name)

	buf := make([]byte, 1024)

	// Receive UDP packets and forward them over transport connections forever
	for {
		n, addr, err := conn.ReadFromUDP(buf)
		fmt.Println("Received ", string(buf[0:n]), " from ", addr)

		if err != nil {
			fmt.Println("Error: ", err)
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
				//ignoring failed writes because packets can be dropped
				_, _ = state.Conn.Write(buf)
			}
		} else {
			// There is not an open transport connection and a connection attempt is not in progress.
			// Open a transport connection.

			fmt.Println("Opening connection to ", target)

			modes.OpenConnection(&tracker, addr.String(), target, name, options, proxyURI)

			// Drop the packet.
			fmt.Println("recv: Open")
		}
	}
}

func ServerSetup(ptServerInfo pt.ServerInfo, stateDir string, options string) (launched bool) {
	return modes.ServerSetupUDP(ptServerInfo, stateDir, options, serverHandler)
}

func serverHandler(name string, remote net.Conn, info *pt.ServerInfo) {
	var header *common.Message

	addrStr := log.ElideAddr(remote.RemoteAddr().String())
	fmt.Println("### handling", name)
	log.Infof("%s(%s) - new connection", name, addrStr)

	serverAddr, err := net.ResolveUDPAddr("udp", info.OrAddr.String())
	if err != nil {
		_ = remote.Close()

		golog.Fatal(err)
	}

	localAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		_ = remote.Close()
		golog.Fatal(err)
	}

	dest, err := net.DialUDP("udp", localAddr, serverAddr)
	if err != nil {
		_ = remote.Close()
		golog.Fatal(err)
	}

	fmt.Println("pumping")

	headerBuffer := make([]byte, 20)

	for {
		fmt.Println("reading...")
		// Read the incoming connection into the buffer.
		_, err := io.ReadFull(remote, headerBuffer)
		if err != nil {
			fmt.Println("read error")
			break
		}

		header, err = goturn.ParseStun(headerBuffer)
		if err != nil {
			fmt.Println("parse error")
			break
		}

		fmt.Println(header.Length)

		fmt.Println("reading data")

		readBuffer := make([]byte, header.Length)
		_, err = io.ReadFull(remote, readBuffer)
		if err != nil {
			fmt.Println("read error")
			break
		}

		writeBuffer := append(headerBuffer, readBuffer...)

		_, _ = dest.Write(writeBuffer)
	}

	_ = dest.Close()
	_ = remote.Close()
}
