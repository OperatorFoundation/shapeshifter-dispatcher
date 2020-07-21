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
package transparent_udp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/OperatorFoundation/shapeshifter-dispatcher/common/log"
	"github.com/OperatorFoundation/shapeshifter-dispatcher/modes"
	"github.com/OperatorFoundation/shapeshifter-ipc/v2"

	"io"
	golog "log"
	"net"
	"net/url"
)

func ClientSetup(socksAddr string, target string, ptClientProxy *url.URL, names []string, options string) bool {
	return modes.ClientSetupUDP(socksAddr, target, ptClientProxy, names, options, clientHandler)
}

func clientHandler(target string, name string, options string, conn *net.UDPConn, proxyURI *url.URL) {
	var length16 uint16

	tracker := make(modes.ConnTracker)

	buf := make([]byte, 1024)

	// Receive UDP packets and forward them over transport connections forever
	for {
		numBytes, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Error: ", err)
		}

		goodBytes := buf[:numBytes]

		fmt.Println(tracker)

		if state, ok := tracker[addr.String()]; ok {
			// There is an open transport connection, or a connection attempt is in progress.

			if state.Waiting {
				// The connection attempt is in progress.
				// Drop the packet.
			} else {
				// There is an open transport connection.
				// Send the packet through the transport.
				length16 = uint16(numBytes)
				lengthBuf := new(bytes.Buffer)
				err = binary.Write(lengthBuf, binary.LittleEndian, length16)
				if err != nil {
					fmt.Println("binary.Write failed:", err)
				} else {
					println("writing data to server")
					println(len(lengthBuf.Bytes()))
					_, writErr := state.Conn.Write(lengthBuf.Bytes())
					if writErr != nil {
						continue
					} else {
						println("writing data to server")
						println(len(goodBytes))
						_, writeBufErr := state.Conn.Write(goodBytes)
						if writeBufErr != nil {
							_ = state.Conn.Close()
							_ = conn.Close()
						}

					}
				}
			}
		} else {
			// There is not an open transport connection and a connection attempt is not in progress.
			// Open a transport connection.

			modes.OpenConnection(&tracker, addr.String(), target, name, options, proxyURI)

			// Drop the packet.
		}
	}
}

func ServerSetup(ptServerInfo pt.ServerInfo, stateDir string, options string) (launched bool) {
	return modes.ServerSetupUDP(ptServerInfo, stateDir, options, serverHandler)
}

func serverHandler(name string, remote net.Conn, info *pt.ServerInfo) {
	var length16 uint16

	addrStr := log.ElideAddr(remote.RemoteAddr().String())
	fmt.Println("### handling", name)
	log.Infof("%s(%s) - new connection", name, addrStr)

	serverAddr, err := net.ResolveUDPAddr("udp", info.OrAddr.String())
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

	lengthBuffer := make([]byte, 2)

	for {
		fmt.Println("reading...")
		// Read the incoming connection into the buffer.
		readLen, err := io.ReadFull(remote, lengthBuffer)
		if err != nil {
			fmt.Println("read error")
			break
		}

		fmt.Println(readLen)

		err = binary.Read(bytes.NewReader(lengthBuffer), binary.LittleEndian, &length16)
		if err != nil {
			fmt.Println("deserialization error")
			_ = dest.Close()
			return
		}

		fmt.Println("reading data")
		fmt.Println(length16)
		readBuffer := make([]byte, length16)
		readLen, err = io.ReadFull(remote, readBuffer)
		if err != nil {
			fmt.Println("read error")
			break
		}
		if readLen != int(length16) {
			println("short read")
			break
		}
		_, _ = dest.Write(readBuffer)
	}

	_ = dest.Close()
}
