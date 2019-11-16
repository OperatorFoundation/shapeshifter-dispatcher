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
	options2 "github.com/OperatorFoundation/shapeshifter-dispatcher/common"
	"github.com/OperatorFoundation/shapeshifter-dispatcher/common/log"
	"github.com/OperatorFoundation/shapeshifter-dispatcher/common/pt_extras"
	"github.com/OperatorFoundation/shapeshifter-dispatcher/transports"
	"github.com/OperatorFoundation/shapeshifter-ipc"
	"github.com/OperatorFoundation/shapeshifter-transports/transports/Dust"
	replicant "github.com/OperatorFoundation/shapeshifter-transports/transports/Replicant"
	"github.com/OperatorFoundation/shapeshifter-transports/transports/meeklite"
	"github.com/OperatorFoundation/shapeshifter-transports/transports/obfs2"
	"github.com/OperatorFoundation/shapeshifter-transports/transports/obfs4"
	"github.com/OperatorFoundation/shapeshifter-transports/transports/shadow"
	"golang.org/x/net/proxy"
	"io"
	golog "log"
	"net"
	"net/url"
	//"github.com/OperatorFoundation/shapeshifter-transports/transports/Optimizer"
	//"github.com/OperatorFoundation/shapeshifter-transports/transports/shadow"
)

type ConnState struct {
	Conn    net.Conn
	Waiting bool
}

func NewConnState() ConnState {
	return ConnState{nil, true}
}

type ConnTracker map[string]ConnState

func ClientSetup(socksAddr string, target string, ptClientProxy *url.URL, names []string, options string) bool {
	// Launch each of the client listeners.
	for _, name := range names {
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

		go clientHandler(target, name, options, ln, ptClientProxy)

		log.Infof("%s - registered listener", name)
	}

	return true
}

func clientHandler(target string, name string, options string, conn *net.UDPConn, proxyURI *url.URL) {
	var length16 uint16

	defer conn.Close()

	fmt.Println("@@@ handling...")

	tracker := make(ConnTracker)

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
				length16 = uint16(n)
				lengthBuf := new(bytes.Buffer)
				err = binary.Write(lengthBuf, binary.LittleEndian, length16)
				if err != nil {
					fmt.Println("binary.Write failed:", err)
				} else {
					fmt.Println("writing...")
					fmt.Println(length16)
					fmt.Println(lengthBuf.Bytes())
					_, writErr := state.Conn.Write(lengthBuf.Bytes())
					if writErr != nil {
						continue
					} else {
						_, writeBufErr := state.Conn.Write(buf)
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

			fmt.Println("Opening connection to ", target)

			openConnection(&tracker, addr.String(), target, name, options, proxyURI)

			// Drop the packet.
			fmt.Println("recv: Open")
		}
	}
}

func openConnection(tracker *ConnTracker, addr string, target string, name string, options string, proxyURI *url.URL) {
	fmt.Println("Making dialer...")

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

	args, argsErr := options2.ParseOptions(options)
	if argsErr != nil {
		log.Errorf("Error parsing transport options: %s", options)
		return
	}
	// Deal with arguments.
	transport, argsToDialerErr := pt_extras.ArgsToDialer(target, name, args, dialer)
	if argsToDialerErr != nil {
		log.Errorf("Error creating a transport with the provided options: %s", options)
		log.Errorf("Error: %s", argsToDialerErr)
		return
	}
	fmt.Println("Dialing ", target)
	remote, _ := transport.Dial()
	// if err != nil {
	// 	fmt.Println("outgoing connection failed", err)
	// 	log.Errorf("(%s) - outgoing connection failed: %s", target, log.ElideError(err))
	// 	fmt.Println("Failed")
	// 	delete(*tracker, addr)
	// 	return
	// }

	fmt.Println("Success")

	(*tracker)[addr] = ConnState{remote, false}
}

func ServerSetup(ptServerInfo pt.ServerInfo, stateDir string, options string) (launched bool, listeners []net.Listener) {
	fmt.Println("ServerSetup")

	// Launch each of the server listeners.
	for _, bindaddr := range ptServerInfo.Bindaddrs {
		name := bindaddr.MethodName
		fmt.Println("bindaddr", bindaddr)

		var listen func(address string) net.Listener

		args, argsErr := options2.ParseServerOptions(options)
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
			transport, err := obfs4.NewObfs4Server(stateDir)
			if err != nil {
				log.Errorf("Can't start obfs4 transport: %v", err)
				return false, nil
			}
			listen = transport.Listen
		case "Replicant":
			shargs, aok := args["Replicant"]
			if !aok {
				return false, nil
			}

			config, err := transports.ParseReplicantConfig(shargs)
			if err != nil {
				return false, nil
			}
			transport := replicant.New(*config)
			listen = transport.Listen
		case "Dust":
			shargs, aok := args["Dust"]
			if !aok {
				return false, nil
			}

			untypedIdPath, ok := shargs["Url"]
			if !ok {
				return false, nil
			}
			idPath, err := options2.CoerceToString(untypedIdPath)
			if err != nil {
				log.Errorf("could not coerce Dust Url to string")
				return false, nil
			}
			transport := Dust.NewDustServer(idPath)
			listen = transport.Listen
		case "meeklite":
			args, aok := args["meeklite"]
			if !aok {
				return false, nil
			}

			untypedUrl, ok := args["Url"]
			if !ok {
				return false, nil
			}

			Url, err := options2.CoerceToString(untypedUrl)
			if err != nil {
				log.Errorf("could not coerce meeklite Url to string")
			}

			untypedFront, ok := args["front"]
			if !ok {
				return false, nil
			}

			front, err2 := options2.CoerceToString(untypedFront)
			if err2 != nil {
				log.Errorf("could not coerce meeklite front to string")
			}

			transport := meeklite.NewMeekTransportWithFront(Url, front)
			listen = transport.Listen
		case "shadow":
			args, aok := args["shadow"]
			if !aok {
				return false, nil
			}

			untypedPassword, ok := args["password"]
			if !ok {
				return false, nil
			}

			Password, err := options2.CoerceToString(untypedPassword)
			if err != nil {
				log.Errorf("could not coerce shadow password to string")
			}

			untypedCertString, ok := args["certString"]
			if !ok {
				return false, nil
			}

			certString, err2 := options2.CoerceToString(untypedCertString)
			if err2 != nil {
				log.Errorf("could not coerce shadow certString to string")
			}

			transport := shadow.NewShadowServer(Password, certString)
			listen = transport.Listen
		default:
			log.Errorf("Unknown transport: %s", name)
			return
		}

		f := listen

		transportLn := f(bindaddr.Addr.String())

		go serverAcceptLoop(name, transportLn, &ptServerInfo)

		log.Infof("%s - registered listener: %s", name, log.ElideAddr(bindaddr.Addr.String()))

		listeners = append(listeners, transportLn)
		launched = true
	}

	return
}

// Resolve an address string into a net.TCPAddr. We are a bit more strict than
// net.ResolveTCPAddr; we don't allow an empty host or port, and the host part
// must be a literal IP address.
//func resolveAddr(addrStr string) (*net.TCPAddr, error) {
//	ipStr, portStr, err := net.SplitHostPort(addrStr)
//	if err != nil {
//		// Before the fixing of bug #7011, tor doesn't put brackets around IPv6
//		// addresses. Split after the last colon, assuming it is a port
//		// separator, and try adding the brackets.
//		parts := strings.Split(addrStr, ":")
//		if len(parts) <= 2 {
//			return nil, err
//		}
//		addrStr := "[" + strings.Join(parts[:len(parts)-1], ":") + "]:" + parts[len(parts)-1]
//		ipStr, portStr, err = net.SplitHostPort(addrStr)
//	}
//	if err != nil {
//		return nil, err
//	}
//	if ipStr == "" {
//		return nil, net.InvalidAddrError(fmt.Sprintf("address string %q lacks a host part", addrStr))
//	}
//	if portStr == "" {
//		return nil, net.InvalidAddrError(fmt.Sprintf("address string %q lacks a port part", addrStr))
//	}
//	ip := net.ParseIP(ipStr)
//	if ip == nil {
//		return nil, net.InvalidAddrError(fmt.Sprintf("not an IP string: %q", ipStr))
//	}
//	port, err := parsePort(portStr)
//	if err != nil {
//		return nil, err
//	}
//	return &net.TCPAddr{IP: ip, Port: port}, nil
//}
//
//func parsePort(portStr string) (int, error) {
//	port, err := strconv.ParseUint(portStr, 10, 16)
//	return int(port), err
//}

func serverAcceptLoop(name string, ln net.Listener, info *pt.ServerInfo) {
	for {
		conn, err := ln.Accept()
		fmt.Println("accepted")
		if err != nil {
			if e, ok := err.(net.Error); ok && !e.Temporary() {
				log.Errorf("serverAcceptLoop failed")
				_ = ln.Close()
				return
			}
			continue
		}
		go serverHandler(name, conn, info)
	}
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

	defer dest.Close()

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
			return
		}

		fmt.Println("reading data")

		readBuffer := make([]byte, length16)
		readLen, err = io.ReadFull(remote, readBuffer)
		if err != nil {
			fmt.Println("read error")
			break
		}

		_, _ = dest.Write(readBuffer)
	}
}
