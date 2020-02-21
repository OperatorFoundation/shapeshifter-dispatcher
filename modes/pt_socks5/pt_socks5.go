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
	"encoding/json"
	"fmt"
	options2 "github.com/OperatorFoundation/shapeshifter-dispatcher/common"
	"github.com/OperatorFoundation/shapeshifter-dispatcher/common/pt_extras"
	"golang.org/x/net/proxy"
	"io"
	"net"
	"net/url"
	"sync"

	"github.com/OperatorFoundation/shapeshifter-dispatcher/common/log"
	"github.com/OperatorFoundation/shapeshifter-dispatcher/common/socks5"
	"github.com/OperatorFoundation/shapeshifter-ipc"

	"github.com/OperatorFoundation/shapeshifter-transports/transports/Dust/v2"
	"github.com/OperatorFoundation/shapeshifter-transports/transports/meeklite/v2"
	"github.com/OperatorFoundation/shapeshifter-transports/transports/obfs2/v2"
	"github.com/OperatorFoundation/shapeshifter-transports/transports/obfs4/v2"
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

		log.Infof("%s - registered listener: %s", name, ln.Addr())

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
				log.Errorf("serverAcceptLoop failed")
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
		log.Errorf("%s - client failed socks handshake: %s", name, err)
		return
	}
	addrStr := log.ElideAddr(socksReq.Target)

	//var args pt.Args
	//if needOptions {
	//	args = socksReq.Args
	//} else {
	//	args, err = pt.ParsePT2ClientParameters(options)
	//	if err != nil {
	//		return
	//	}
	//}

	//args, argsErr := options2.ParseOptions(options)
	//if argsErr != nil {
	//	log.Errorf("Error parsing transport options: %s", options)
	//	return
	//}

	var dialer proxy.Dialer

	// Deal with arguments.
	transport, argsToDialerErr := pt_extras.ArgsToDialer(socksReq.Target, name, options, dialer)
	if argsToDialerErr != nil {
		log.Errorf("Error creating a transport with the provided options: %s", options)
		log.Errorf("Error: %s", argsToDialerErr)
		return
	}
	// Obtain the proxy dialer if any, and create the outgoing TCP connection.
	if proxyURI != nil {
		var proxyErr error
		dialer, proxyErr = proxy.FromURL(proxyURI, proxy.Direct)
		if proxyErr != nil {
			// This should basically never happen, since config protocol
			// verifies this.
			log.Errorf("%s(%s) - failed to obtain proxy dialer: %s", name, addrStr, log.ElideError(err))
			_ = socksReq.Reply(socks5.ReplyGeneralFailure)
			return
		}
	}

	fmt.Println("Got dialer", dialer, proxyURI, proxy.Direct)

	remote, err2 := transport.Dial()
	if err2 != nil {
		log.Errorf("%s(%s) - outgoing connection failed: %s", name, addrStr, log.ElideError(err))
		_ = socksReq.Reply(socks5.ErrorToReplyCode(err))
		return
	}
	err = socksReq.Reply(socks5.ReplySucceeded)
	if err != nil {
		log.Errorf("%s(%s) - SOCKS reply failed: %s", name, addrStr, log.ElideError(err))
		return
	}

	if err = copyLoop(conn, remote); err != nil {
		log.Warnf("%s(%s) - closed connection: %s", name, addrStr, log.ElideError(err))
	} else {
		log.Infof("%s(%s) - closed connection", name, addrStr)
	}

	return
}

func ServerSetup(ptServerInfo pt.ServerInfo, statedir string, options string) (launched bool) {
	for _, bindaddr := range ptServerInfo.Bindaddrs {
		name := bindaddr.MethodName

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
			transport, err := obfs4.NewObfs4Server(statedir)
			if err != nil {
				log.Errorf("Can't start obfs4 transport: %v", err)
				return
			}

			listen = transport.Listen
		case "replicant":
			shargs, aok := args["Replicant"]
			if !aok {
				return false
			}

			shargsBytes, err:= json.Marshal(shargs)
			shargsString := string(shargsBytes)
			config, err := transports.ParseArgsReplicantServer(shargsString)
			if err != nil {
				return false
			}

			//FIXME: Correct address for listen
			config.Listen(bindaddr.Addr.String())
			//var dialer proxy.Dialer
			//transport := replicant.New(*config, dialer)
			//listen = transport.Listen
		case "Dust":
			shargs, aok := args["Dust"]
			if !aok {
				return false
			}

			untypedIdPath, ok := shargs["Url"]
			if !ok {
				return false
			}
			idPathByte, err:= json.Marshal(untypedIdPath)
			idPathString := string(idPathByte)
			if err != nil {
				log.Errorf("could not coerce Dust Url to string")
				return false
			}
			transport := Dust.NewDustServer(idPathString)
			listen = transport.Listen
		case "meeklite":
			args, aok := args["meeklite"]
			if !aok {
				return false
			}

			untypedUrl, ok := args["Url"]
			if !ok {
				return false
			}

			UrlByte, err:= json.Marshal(untypedUrl)
			UrlString := string(UrlByte)
			if err != nil {
				log.Errorf("could not coerce meeklite Url to string")
			}

			untypedFront, ok := args["front"]
			if !ok {
				return false
			}

			FrontByte, err2:= json.Marshal(untypedFront)
			FrontString := string(FrontByte)
			if err2 != nil {
				log.Errorf("could not coerce meeklite front to string")
			}
			var dialer proxy.Dialer
			transport := meeklite.NewMeekTransportWithFront(UrlString, FrontString, dialer)
			listen = transport.Listen
		case "shadow":
			args, aok := args["shadow"]
			if !aok {
				return false
			}

			argsBytes, err:= json.Marshal(args)
			argsString := string(argsBytes)
			config, err := transports.ParseArgsShadowServer(argsString)
			if err != nil {
				println("Received a Replicant config error: ", err.Error())
				log.Errorf(err.Error())
				return false
			}

			listen = config.Listen
		default:
			log.Errorf("Unknown transport: %s", name)
			return
		}

		// if args := f.Args(); args != nil {
		// 	pt.SmethodArgs(name, ln.Addr(), *args)
		// } else {
		// 	pt.SmethodArgs(name, ln.Addr(), nil)
		// }

		go func() {
			for {
				transportLn := listen(bindaddr.Addr.String())
				log.Infof("%s - registered listener: %s", name, log.ElideAddr(bindaddr.Addr.String()))
				serverAcceptLoop(name, transportLn, &ptServerInfo)
				transportLn.Close()
			}
		}()

		launched = true
	}
	pt.SmethodsDone()

	return
}

func serverAcceptLoop(name string, ln net.Listener, info *pt.ServerInfo) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			if e, ok := err.(net.Error); ok && !e.Temporary() {
				return
			}
			continue
		}
		go serverHandler(name, conn, info)
	}
}

func serverHandler(name string, remote net.Conn, info *pt.ServerInfo) {

	addrStr := log.ElideAddr(remote.RemoteAddr().String())
	log.Infof("%s(%s) - new connection", name, addrStr)

	// Connect to the orport.
	orConn, err := pt.DialOr(info, remote.RemoteAddr().String(), name)
	if err != nil {
		log.Errorf("%s(%s) - failed to connect to ORPort: %s", name, addrStr, log.ElideError(err))
		return
	}

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
		_, err := io.Copy(b, a)
		errChan <- err
	}()
	go func() {
		defer wg.Done()
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
