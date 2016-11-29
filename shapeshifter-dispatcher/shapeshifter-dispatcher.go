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
package main

import (
	"flag"
	"fmt"
	golog "log"
	"net"
	"net/url"
	"os"
	"path"
	"strings"
	"syscall"

	"git.torproject.org/pluggable-transports/goptlib.git"
	"github.com/OperatorFoundation/shapeshifter-dispatcher/common/log"
	"github.com/OperatorFoundation/shapeshifter-dispatcher/common/pt_extras"
	"github.com/OperatorFoundation/shapeshifter-dispatcher/common/termmon"

	"github.com/OperatorFoundation/shapeshifter-dispatcher/modes/pt_socks5"
	"github.com/OperatorFoundation/shapeshifter-dispatcher/modes/stun_udp"
	"github.com/OperatorFoundation/shapeshifter-dispatcher/modes/transparent_tcp"
	"github.com/OperatorFoundation/shapeshifter-dispatcher/modes/transparent_udp"

	_ "github.com/OperatorFoundation/obfs4/proxy_dialers/proxy_http"
	_ "github.com/OperatorFoundation/obfs4/proxy_dialers/proxy_socks4"
	"github.com/OperatorFoundation/shapeshifter-dispatcher/transports"
	"github.com/OperatorFoundation/shapeshifter-transports/transports/base"
)

const (
	dispatcherVersion = "0.0.7-dev"
	dispatcherLogFile = "dispatcher.log"
	socksAddr         = "127.0.0.1:0"
)

var stateDir string
var termMon *termmon.TermMonitor

func getVersion() string {
	return fmt.Sprintf("dispatcher-%s", dispatcherVersion)
}

func main() {
	// Initialize the termination state monitor as soon as possible.
	termMon = termmon.NewTermMonitor()

	// Handle the command line arguments.
	_, execName := path.Split(os.Args[0])

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "shapeshifter-dispatcher is a PT v2.0 proxy supporting multiple transports and proxy modes\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n\t%s --client --state [statedir] --ptversion 2 --transports [transport1,transport2,...]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Example:\n\t%s --client --state state --ptversion 2 --transports obfs2\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Flags:\n\n")
		flag.PrintDefaults()
	}

	// PT 2.0 specification, 3.3.1.1. Common Configuration Parameters
	// FIXME: in the spec, this is -version, which is already used for printing the version number
	ptversion := flag.String("ptversion", "", "Specify the Pluggable Transport protocol version to use")
	statePath := flag.String("state", "", "Specify the directory to use to store state information required by the transports")
	// FIXME: -exit-on-stdin-close

	// NOTE: -transports is parsed as a common command line flag that overrides either TOR_PT_SERVER_TRANSPORTS or TOR_PT_CLIENT_TRANSPORTS
	transportsList := flag.String("transports", "", "Specify transports to enable")

	// PT 2.0 specification, 3.3.1.2. Pluggable PT Client Configuration Parameters
	proxy := flag.String("proxy", "", "Specify an HTTP or SOCKS4a proxy that the PT needs to use to reach the Internet")

	// PT 2.0 specification, 3.3.1.3. Pluggable PT Server Environment Variables
	// FIXME: -options
	bindAddr := flag.String("bindaddr", "", "Specify the bind address for transparent server")
	// FIXME: -orport
	// FIXME: -extorport
	// FIXME: -authcookie

	// Additional command line flags inherited from obfs4proxy
	showVer := flag.Bool("version", false, "Print version and exit")
	logLevelStr := flag.String("logLevel", "ERROR", "Log level (ERROR/WARN/INFO/DEBUG)")
	enableLogging := flag.Bool("enableLogging", false, "Log to TOR_PT_STATE_LOCATION/"+dispatcherLogFile)
	unsafeLogging := flag.Bool("unsafeLogging", false, "Disable the address scrubber")

	// Additional command line flags added to shapeshifter-dispatcher
	clientMode := flag.Bool("client", false, "Enable client mode")
	serverMode := flag.Bool("server", false, "Enable server mode")
	transparent := flag.Bool("transparent", false, "Enable transparent proxy mode. The default is protocol-aware proxy mode (SOCKS5 for TCP, STUN for UDP)")
	udp := flag.Bool("udp", false, "Enable UDP proxy mode. The default is TCP proxy mode.")
	target := flag.String("target", "", "Specify transport server destination address")
	flag.Parse()

	if *showVer {
		fmt.Printf("%s\n", getVersion())
		os.Exit(0)
	}
	if err := log.SetLogLevel(*logLevelStr); err != nil {
		fmt.Println("failed to set log level")
		golog.Fatalf("[ERROR]: %s - failed to set log level: %s", execName, err)
	}

	// Determine if this is a client or server, initialize the common state.
	var ptListeners []net.Listener
	launched := false
	isClient, err := checkIsClient(*clientMode, *serverMode)
	if err != nil {
		flag.Usage()
		golog.Fatalf("[ERROR]: %s - either --client or --server is required, or configure using PT 2.0 environment variables", execName)
	}
	if stateDir, err = makeStateDir(*statePath); err != nil {
		flag.Usage()
		golog.Fatalf("[ERROR]: %s - No state directory: Use --state or TOR_PT_STATE_LOCATION environment variable", execName)
	}
	if err = log.Init(*enableLogging, path.Join(stateDir, dispatcherLogFile), *unsafeLogging); err != nil {
		golog.Fatalf("[ERROR]: %s - failed to initialize logging", execName)
	}
	if err = transports.Init(); err != nil {
		log.Errorf("%s - failed to initialize transports: %s", execName, err)
		os.Exit(-1)
	}

	log.Noticef("%s - launched", getVersion())
	fmt.Println("launching")

	if *transparent {
		// Do the transparent proxy configuration.
		log.Infof("%s - initializing transparent proxy", execName)
		if *udp {
			log.Infof("%s - initializing UDP transparent proxy", execName)
			if isClient {
				log.Infof("%s - initializing client transport listeners", execName)
				if *target == "" {
					log.Errorf("%s - transparent mode requires a target", execName)
				} else {
					fmt.Println("transparent udp client")
					factories, ptClientProxy := getClientFactories(ptversion, transportsList, proxy)
					launched = transparent_udp.ClientSetup(termMon, *target, factories, ptClientProxy)
				}
			} else {
				log.Infof("%s - initializing server transport listeners", execName)
				if *bindAddr == "" {
					fmt.Println("%s - transparent mode requires a bindaddr", execName)
				} else {
					fmt.Println("transparent udp server")
					launched = transparent_udp.ServerSetup(termMon, *bindAddr, *target)
					fmt.Println("launched", launched, ptListeners)
				}
			}
		} else {
			log.Infof("%s - initializing TCP transparent proxy", execName)
			if isClient {
				log.Infof("%s - initializing client transport listeners", execName)
				if *target == "" {
					log.Errorf("%s - transparent mode requires a target", execName)
				} else {
					factories, ptClientProxy := getClientFactories(ptversion, transportsList, proxy)
					launched, ptListeners = transparent_tcp.ClientSetup(termMon, *target, factories, ptClientProxy)
				}
			} else {
				log.Infof("%s - initializing server transport listeners", execName)
				if *bindAddr == "" {
					fmt.Println("%s - transparent mode requires a bindaddr", execName)
				} else {
					launched, ptListeners = transparent_tcp.ServerSetup(termMon, *bindAddr)
					fmt.Println("launched", launched, ptListeners)
				}
			}
		}
	} else {
		if *udp {
			log.Infof("%s - initializing STUN UDP proxy", execName)
			if isClient {
				log.Infof("%s - initializing client transport listeners", execName)
				if *target == "" {
					log.Errorf("%s - STUN mode requires a target", execName)
				} else {
					fmt.Println("STUN udp client")
					factories, ptClientProxy := getClientFactories(ptversion, transportsList, proxy)
					launched = stun_udp.ClientSetup(termMon, *target, factories, ptClientProxy)
				}
			} else {
				log.Infof("%s - initializing server transport listeners", execName)
				if *bindAddr == "" {
					fmt.Println("%s - STUN mode requires a bindaddr", execName)
				} else {
					fmt.Println("STUN udp server")
					launched = stun_udp.ServerSetup(termMon, *bindAddr, *target)
					fmt.Println("launched", launched, ptListeners)
				}
			}
		} else {
			// Do the managed pluggable transport protocol configuration.
			log.Infof("%s - initializing PT 1.0 proxy", execName)
			if isClient {
				log.Infof("%s - initializing client transport listeners", execName)
				factories, ptClientProxy := getClientFactories(ptversion, transportsList, proxy)
				launched, ptListeners = pt_socks5.ClientSetup(termMon, factories, ptClientProxy)
			} else {
				log.Infof("%s - initializing server transport listeners", execName)
				launched, ptListeners = pt_socks5.ServerSetup(termMon)
			}
		}
	}

	if !launched {
		// Initialization failed, the client or server setup routines should
		// have logged, so just exit here.
		os.Exit(-1)
	}

	fmt.Println("launched")

	log.Infof("%s - accepting connections", execName)
	defer func() {
		log.Noticef("%s - terminated", execName)
	}()

	// At this point, the pt config protocol is finished, and incoming
	// connections will be processed.  Wait till the parent dies
	// (immediate exit), a SIGTERM is received (immediate exit),
	// or a SIGINT is received.
	if sig := termMon.Wait(false); sig == syscall.SIGTERM {
		return
	}

	// Ok, it was the first SIGINT, close all listeners, and wait till,
	// the parent dies, all the current connections are closed, or either
	// a SIGINT/SIGTERM is received, and exit.
	for _, ln := range ptListeners {
		ln.Close()
	}

	termMon.Wait(true)

	fmt.Println("waiting")
	for {
		// FIXME - block because termMon.Wait is not blocking
	}
}

func checkIsClient(client bool, server bool) (bool, error) {
	if client {
		return true, nil
	} else if server {
		return false, nil
	} else {
		return pt_extras.PtIsClient()
	}
}

func makeStateDir(statePath string) (string, error) {
	if statePath != "" {
		err := os.MkdirAll(statePath, 0700)
		return statePath, err
	} else {
		return pt.MakeStateDir()
	}
}

func getClientFactories(ptversion *string, transportsList *string, proxy *string) (clientProxy *url.URL, factories map[string]base.ClientFactory) {
	var ptClientInfo pt.ClientInfo
	var err error

	// FIXME - instead of this, goptlib should be modified to accept command line flag override of EITHER ptversion or transports (or both)
	if ptversion == nil || transportsList == nil {
		fmt.Println("Falling back to environment variables for ptversion/transports", ptversion, transportsList)
		ptClientInfo, err = pt.ClientSetup(transports.Transports())
		if err != nil {
			// FIXME - print a more useful error, specifying --ptversion and --transports flags
			golog.Fatal(err)
		}
	} else {
		if *transportsList == "*" {
			ptClientInfo = pt.ClientInfo{MethodNames: transports.Transports()}
		} else {
			ptClientInfo = pt.ClientInfo{MethodNames: strings.Split(*transportsList, ",")}
		}
	}

	ptClientProxy, err := pt_extras.PtGetProxy(proxy)
	fmt.Println("ptclientproxy", ptClientProxy)
	if err != nil {
		golog.Fatal(err)
	} else if ptClientProxy != nil {
		pt_extras.PtProxyDone()
	}

	factories = make(map[string]base.ClientFactory)

	// Launch each of the client listeners.
	for _, name := range ptClientInfo.MethodNames {
		t := transports.Get(name)
		if t == nil {
			pt.CmethodError(name, "no such transport is supported")
			continue
		}

		f, err := t.ClientFactory(stateDir)
		if err != nil {
			pt.CmethodError(name, "failed to get ClientFactory")
			continue
		}

		factories[name] = f
	}

	return ptClientProxy, factories
}
