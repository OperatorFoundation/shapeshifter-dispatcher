/*
 * Copyright (c) 2014, Yawning Angel <yawning at torproject dot org>
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

package pt_extras

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
	"io"
	"os"

)

// This file contains things that probably should be in goptlib but are not
// yet or are not finalized.

func ptProxyError(msg string) error {
	line := []byte(fmt.Sprintf("PROXY-ERROR %s\n", msg))
	_, _ = io.Writer.Write(syncWriter{os.Stdout}, line)
	return errors.New(msg)
}

func PtProxyDone() {
	line := []byte("PROXY DONE\n")
	_, _ = io.Writer.Write(syncWriter{os.Stdout}, line)
}

func PtGetProxy(proxy *string) (*url.URL, error) {
	var specString string

	if proxy != nil {
		specString = *proxy
	} else {
		return nil, errors.New("no proxy specified")
	}
	if specString == "" {
		return nil, nil
	}
	spec, err := url.Parse(specString)
	if err != nil {
		return nil, ptProxyError(fmt.Sprintf("failed to parse proxy config: %s", err))
	}

	if !spec.IsAbs() {
		return nil, ptProxyError("proxy URI is relative, must be absolute")
	}
	if spec.Path != "" {
		return nil, ptProxyError("proxy URI has a path defined")
	}
	if spec.RawQuery != "" {
		return nil, ptProxyError("proxy URI has a query defined")
	}
	if spec.Fragment != "" {
		return nil, ptProxyError("proxy URI has a fragment defined")
	}

	switch spec.Scheme {
	case "http":
		// The most forgiving of proxies.

	case "socks4a":
		if spec.User != nil {
			_, isSet := spec.User.Password()
			if isSet {
				return nil, ptProxyError("proxy URI specified SOCKS4a and a password")
			}
		}

	case "socks5":
		if spec.User != nil {
			// USER/PASSWD both must be between 1 and 255 bytes long. (RFC1929)
			user := spec.User.Username()
			passwd, isSet := spec.User.Password()
			if len(user) < 1 || len(user) > 255 {
				return nil, ptProxyError("proxy URI specified a invalid SOCKS5 username")
			}
			if !isSet || len(passwd) < 1 || len(passwd) > 255 {
				return nil, ptProxyError("proxy URI specified a invalid SOCKS5 password")
			}
		}

	default:
		return nil, ptProxyError(fmt.Sprintf("proxy URI has invalid scheme: %s", spec.Scheme))
	}

	_, err = resolveAddrStr(spec.Host)
	if err != nil {
		return nil, ptProxyError(fmt.Sprintf("proxy URI has invalid host: %s", err))
	}

	return spec, nil
}

// Sigh, pt.resolveAddr() isn't exported.  Include our own ghetto version that
// doesn't work around #7011, because we don't work with pre-0.2.5.x tor, and
// all we care about is validation anyway.
func resolveAddrStr(addrStr string) (*net.TCPAddr, error) {
	ipStr, portStr, err := net.SplitHostPort(addrStr)
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
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return nil, net.InvalidAddrError(fmt.Sprintf("not a Port string: %q", portStr))
	}

	return &net.TCPAddr{IP: ip, Port: int(port), Zone: ""}, nil
}

type ClientInfo struct {
	MethodNames []string
	ProxyURL    *url.URL
}

type ServerInfo struct {
	Bindaddrs      []Bindaddr
	OrAddr         *net.TCPAddr
	ExtendedOrAddr *net.TCPAddr
	AuthCookiePath string
}

type Bindaddr struct {
	MethodName string
	Addr       *net.TCPAddr
	Options string
}

// Resolve an address string into a net.TCPAddr. We are a bit more strict than
// net.ResolveTCPAddr; we don't allow an empty host or port, and the host part
// must be a literal IP address.
func ResolveAddr(addrStr string) (*net.TCPAddr, error) {
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

// Return a new slice, the members of which are those members of addrs having a
// MethodName in methodNames.
func FilterBindaddrs(addrs []Bindaddr, methodNames []string) []Bindaddr {
	var result []Bindaddr

	for _, ba := range addrs {
		for _, methodName := range methodNames {
			if ba.MethodName == methodName {
				result = append(result, ba)
				break
			}
		}
	}

	return result
}

func DialOr(info *ServerInfo, addr, methodName string) (*net.TCPConn, error) {
	if info.ExtendedOrAddr == nil || info.AuthCookiePath == "" {
		return net.DialTCP("tcp", nil, info.OrAddr)
	}

	s, err := net.DialTCP("tcp", nil, info.ExtendedOrAddr)
	if err != nil {
		return nil, err
	}
	s.SetDeadline(time.Now().Add(5 * time.Second))
	s.SetDeadline(time.Time{})

	return s, nil
}

func parsePort(portStr string) (int, error) {
	port, err := strconv.ParseUint(portStr, 10, 16)
	return int(port), err
}

type syncWriter struct {
	*os.File
}