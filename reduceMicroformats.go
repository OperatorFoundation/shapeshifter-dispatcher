/*
	MIT License

	Copyright (c) 2020 Operator Foundation

	Permission is hereby granted, free of charge, to any person obtaining a copy
	of this software and associated documentation files (the "Software"), to deal
	in the Software without restriction, including without limitation the rights
	to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
	copies of the Software, and to permit persons to whom the Software is
	furnished to do so, subject to the following conditions:

	The above copyright notice and this permission notice shall be included in all
	copies or substantial portions of the Software.

	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
	IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
	FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
	AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
	LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
	OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
	SOFTWARE.
*/

package main

import (
	"errors"
	"github.com/OperatorFoundation/shapeshifter-dispatcher/common/log"
)

//This is for proposal no.9
//serverBindPort := flag.String("bindport", "", "Specify the bind address port for transparent client")
//clientBindHost := flag.String("bindhost", "", "Specify the bind address host for transparent client")
//bindhost:bindport
//serverBindPort := flag.String("bindport", "", "Specify the bind address port for transparent server")
//serverBindHost := flag.String("bindhost", "", "Specify the bind address host for transparent server")
//targetHost := flag.String("targethost", "", "Specify transport server destination address port")
//targetPort := flag.String("targetport", "", "Specify transport server destination address host")
//proxyListenHost := flag.String("proxylistenhost", "", "Specify the bind address for the local SOCKS server host provided by the client")
//proxyListnePort := flag.String("proxylistenport", "", "Specify the bind address for the local SOCKS server port provided by the client")
//modeName := flag.String("mode", "socks5", "Specify which mode is being used: transparent-TCP, transparent-UDP, socks5, or STUN")
//set transparent or udp to nil

func validateTransports(transport *string, transports *string) error {
	if *transports == "" && *transport == "" {
		return errors.New("you must specify either --transport or --transports")
	}

	if *transports != "" && *transport != "" {
		return errors.New("you cannot specify both --transport and --transports")
	}

	return nil
}

func validateServerBindAddr(transport *string, serverBindHost *string, serverBindPort *string, serverBindAddr *string) error {
	if *serverBindHost == "" && *serverBindAddr == "" {
		return errors.New("you must specify either --bindhost or --bindaddr")
	}

	if *serverBindHost != "" && *serverBindAddr != "" {
		return errors.New("you cannot specify both --bindhost and --bindaddr")
	}

	if (*serverBindHost != "" && *serverBindPort == "") || (*serverBindHost == "" && *serverBindPort != "") {
		return errors.New("you must specify both --bindhost and --bindport (or use --bindaddr)")
	}

	if *serverBindHost != "" && *transport == "" {
		return errors.New("you must specify --transport when you use --bindhost")
	}

	return nil
}

func validateProxyListenAddr(proxyListenHost *string, proxyListenPort *string, proxyListenAddr *string) error {
	if *proxyListenHost == "" && *proxyListenAddr == "" {
		return errors.New("you must specify either --proxylistenhost or --proxylistenaddr")
	}

	if *proxyListenHost != "" && *proxyListenAddr != "" {
		log.Infof("proxylistenhost: %s", *proxyListenHost)
		log.Infof("proxylistenport: %s", *proxyListenPort)
		log.Infof("proxylistenaddr: %s", *proxyListenAddr)
		return errors.New("you cannot specify both --proxylistenhost and --proxylistenaddr")
	}

	if (*proxyListenHost != "" && *proxyListenPort == "") || (*proxyListenHost == "" && *proxyListenPort != "") {
		return errors.New("you must specify both --proxylistenhost and --proxylistenport (or use --proxylistenaddr)")
	}

	return nil
}

func validatetarget(isClient bool, targetHost *string, targetPort *string, targetAddr *string) error {
	if isClient {
		if *targetHost != "" || *targetPort != "" || *targetAddr != "" {
			return errors.New("cannot specify --target, --targethost, or --targetport in client mode")
		}
		return nil
	} else {
		if *targetHost == "" && *targetAddr == "" {
			return errors.New("you must specify either --targethost or --target")
		}

		if *targetHost != "" && *targetAddr != "" {
			return errors.New("you cannot specify both --targethost and --target")
		}

		if (*targetHost != "" && *targetPort == "") || (*targetHost == "" && *targetPort != "") {
			return errors.New("you must specify both --targethost and --targetport (or use --target)")
		}
		return nil
	}
}

func validatetargetSocks5(targetHost *string, targetPort *string, targetAddr *string) error {
	if *targetHost != "" {
		return errors.New("you cannot specify --targethost in socks5 mode")
	}

	if *targetPort != "" {
		return errors.New("you cannot specify --targetport in socks5 mode")
	}

	if *targetAddr != "" {
		return errors.New("you cannot specify --target in socks5 mode")
	}

	return nil
}

func validateMode(mode *string, transparent *bool, udp *bool) error {
	if *mode != "" && *transparent != false {
		return errors.New("cannot specify --mode and --transparent at the same time")
	}

	if *mode != "" && *udp != false {
		return errors.New("cannot specify --mode and --udp at the same time")
	}

	if *mode != "" {
		switch *mode {
		case "socks5":
			return nil
		case "transparent-TCP":
			return nil
		case "transparent-UDP":
			return nil
		case "STUN":
			return nil
		default:
			return errors.New("invalid mode")
		}
	}

	return nil
}