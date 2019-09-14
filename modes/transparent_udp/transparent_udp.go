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
	"github.com/OperatorFoundation/shapeshifter-dispatcher/common/pt_extras"
	"github.com/OperatorFoundation/shapeshifter-transports/transports/Dust"
	"github.com/OperatorFoundation/shapeshifter-transports/transports/meeklite"
	"github.com/OperatorFoundation/shapeshifter-transports/transports/shadow"
	"io"
	golog "log"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/OperatorFoundation/shapeshifter-dispatcher/common/log"
	"github.com/OperatorFoundation/shapeshifter-dispatcher/common/termmon"
	"github.com/OperatorFoundation/shapeshifter-ipc"
	"github.com/OperatorFoundation/shapeshifter-transports/transports/obfs2"
	"github.com/OperatorFoundation/shapeshifter-transports/transports/obfs4"
	//"github.com/OperatorFoundation/shapeshifter-transports/transports/Optimizer"
	//"github.com/OperatorFoundation/shapeshifter-transports/transports/shadow"
)

var stateDir string

type ConnState struct {
	Conn    net.Conn
	Waiting bool
}

func NewConnState() ConnState {
	return ConnState{nil, true}
}

type ConnTracker map[string]ConnState

func ClientSetup(termMon *termmon.TermMonitor, socksAddr string, target string, ptClientProxy *url.URL, names []string, options string) bool {
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

		go clientHandler(target, termMon, name, options, ln, ptClientProxy)

		log.Infof("%s - registered listener: %s", name, ln)
	}

	return true
}

func clientHandler(target string, termMon *termmon.TermMonitor, name string, options string, conn *net.UDPConn, proxyURI *url.URL) {
	var length16 uint16

	defer conn.Close()
	termMon.OnHandlerStart()
	defer termMon.OnHandlerFinish()

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
					state.Conn.Write(lengthBuf.Bytes())
					state.Conn.Write(buf)
				}
			}
		} else {
			// There is not an open transport connection and a connection attempt is not in progress.
			// Open a transport connection.

			fmt.Println("Opening connection to ", target)

			openConnection(&tracker, addr.String(), target, termMon, name, options, proxyURI)

			// Drop the packet.
			fmt.Println("recv: Open")
		}
	}
}

func openConnection(tracker *ConnTracker, addr string, target string, termMon *termmon.TermMonitor, name string, options string, proxyURI *url.URL) {
	fmt.Println("Making dialer...")

	newConn := NewConnState()
	(*tracker)[addr] = newConn

	go dialConn(tracker, addr, target, name, options, proxyURI)
}

func dialConn(tracker *ConnTracker, addr string, target string, name string, options string, proxyURI *url.URL) {
	// Obtain the proxy dialer if any, and create the outgoing TCP connection.
	// dialFn := proxy.Direct.Dial
	// if proxyURI != nil {
	// 	dialer, err := proxy.FromURL(proxyURI, proxy.Direct)
	// 	if err != nil {
	// 		// This should basically never happen, since config protocol
	// 		// verifies this.
	// 		fmt.Println("failed to obtain dialer", proxyURI, proxy.Direct)
	// 		log.Errorf("(%s) - failed to obtain proxy dialer: %s", target, log.ElideError(err))
	// 		return
	// 	}
	// 	dialFn = dialer.Dial
	// }

	fmt.Println("Dialing....")

	args, argsErr := options2.ParseOptions(options)
	if argsErr != nil {
		log.Errorf("Error parsing transport options: %s", options)
		return
	}

	// Deal with arguments.
	transport, _ := pt_extras.ArgsToDialer(target, name, args)
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

func ServerSetup(termMon *termmon.TermMonitor, bindaddrString string, ptServerInfo pt.ServerInfo, options string) (launched bool, listeners []net.Listener) {
	fmt.Println("ServerSetup")

	// Launch each of the server listeners.
	for _, bindaddr := range ptServerInfo.Bindaddrs {
		name := bindaddr.MethodName
		fmt.Println("bindaddr", bindaddr)

		var listen func(address string) net.Listener

		args, argsErr := pt.ParsePT2ClientParameters(options)
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
			if cert, ok := args["cert"]; ok {
				if iatModeStr, ok2 := args["iatMode"]; ok2 {
					iatMode, err := strconv.Atoi(iatModeStr[0])
					if err != nil {
						transport := obfs4.NewObfs4Client(cert[0], iatMode)
						listen = transport.Listen
					} else {
						log.Errorf("obfs4 transport bad iatMode value: %s", iatModeStr)
						return
					}
				} else {
					log.Errorf("obfs4 transport missing cert argument: %s", args)
					return
				}
			} else {
				log.Errorf("obfs4 transport missing cert argument: %s", args)
				return
			}
		//case "Replicant":
		//	Config, ok := args.Get("config")
		//	if !ok {
		//		return false, nil
		//	}
		//
		//	transport := replicant.New(Config)
		//	listen = transport.Listen
		case "Dust":
			idPath, ok := args.Get("idPath")
			if !ok {
				return false, nil
			}

			transport := Dust.NewDustServer(idPath)
			listen = transport.Listen
		case "meeklite":
			Url, ok := args.Get("Url")
			if !ok {
				return false, nil
			}

			Front, ok2 := args.Get("Front")
			if !ok2 {
				return false, nil
			}

			transport := meeklite.NewMeekTransportWithFront(Url, Front)
			listen = transport.Listen
		case "shadow":
			password, ok := args.Get("password")
			if !ok {
				return false, nil
			}

			cipherName, ok2 := args.Get("cipherName")
			if !ok2 {
				return false, nil
			}

			transport := shadow.NewShadowServer(password, cipherName)
			listen = transport.Listen
		default:
			log.Errorf("Unknown transport: %s", name)
			return
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

func serverAcceptLoop(termMon *termmon.TermMonitor, name string, ln net.Listener, info *pt.ServerInfo) error {
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
		go serverHandler(termMon, name, conn, info)
	}
}

func serverHandler(termMon *termmon.TermMonitor, name string, remote net.Conn, info *pt.ServerInfo) {
	var length16 uint16

	defer remote.Close()
	termMon.OnHandlerStart()
	defer termMon.OnHandlerFinish()

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

		dest.Write(readBuffer)
	}
}
