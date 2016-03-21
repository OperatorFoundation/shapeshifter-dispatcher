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
	"os"
	"path"
	"syscall"

	"git.torproject.org/pluggable-transports/goptlib.git"
	"github.com/OperatorFoundation/obfs4/common/log"
	"github.com/OperatorFoundation/obfs4/common/termmon"
	"github.com/OperatorFoundation/obfs4/common/pt_extras"
	"github.com/OperatorFoundation/obfs4/transports"

	"github.com/OperatorFoundation/obfs4/proxies/proxy_socks5"
	"github.com/OperatorFoundation/obfs4/proxies/proxy_transparent"
	"github.com/OperatorFoundation/obfs4/proxies/proxy_transparent_udp"
)

const (
	obfs4proxyVersion = "0.0.7-dev"
	obfs4proxyLogFile = "obfs4proxy.log"
	socksAddr         = "127.0.0.1:0"
)

var stateDir string
var termMon *termmon.TermMonitor

func getVersion() string {
	return fmt.Sprintf("obfs4proxy-%s", obfs4proxyVersion)
}

func main() {
	// Initialize the termination state monitor as soon as possible.
	termMon = termmon.NewTermMonitor()

	// Handle the command line arguments.
	_, execName := path.Split(os.Args[0])
	showVer := flag.Bool("version", false, "Print version and exit")
	logLevelStr := flag.String("logLevel", "ERROR", "Log level (ERROR/WARN/INFO/DEBUG)")
	enableLogging := flag.Bool("enableLogging", false, "Log to TOR_PT_STATE_LOCATION/"+obfs4proxyLogFile)
	unsafeLogging := flag.Bool("unsafeLogging", false, "Disable the address scrubber")
	transparent := flag.Bool("transparent", false, "Enable transparent proxy mode")
	udp := flag.Bool("udp", false, "Enable UDP proxy mode")
	target := flag.String("target", "", "Specify transport server destination address")
	clientMode := flag.Bool("client", false, "Enable client mode")
	serverMode := flag.Bool("server", false, "Enable server mode")
	statePath := flag.String("state", "", "Specify transport server destination address")
	bindAddr := flag.String("bindaddr", "", "Specify the bind address for transparent server")
	flag.Parse()

	if *showVer {
		fmt.Printf("%s\n", getVersion())
		os.Exit(0)
	}
	if err := log.SetLogLevel(*logLevelStr); err != nil {
		golog.Fatalf("[ERROR]: %s - failed to set log level: %s", execName, err)
	}

	// Determine if this is a client or server, initialize the common state.
	var ptListeners []net.Listener
	launched := false
	isClient, err := checkIsClient(*clientMode, *serverMode)
	if err != nil {
		golog.Fatalf("[ERROR]: %s - must be run as a managed transport", execName)
	}
	if stateDir, err = makeStateDir(*statePath); err != nil {
		golog.Fatalf("[ERROR]: %s - No state directory: %s", execName, err)
	}
	if err = log.Init(*enableLogging, path.Join(stateDir, obfs4proxyLogFile), *unsafeLogging); err != nil {
		golog.Fatalf("[ERROR]: %s - failed to initialize logging", execName)
	}
	if err = transports.Init(); err != nil {
		log.Errorf("%s - failed to initialize transports: %s", execName, err)
		os.Exit(-1)
	}

	log.Noticef("%s - launched", getVersion())

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
					launched = proxy_transparent_udp.ClientSetup(termMon, *target)
				}
			} else {
				log.Infof("%s - initializing server transport listeners", execName)
				if *bindAddr == "" {
					fmt.Println("%s - transparent mode requires a bindaddr", execName)
				} else {
					launched = proxy_transparent_udp.ServerSetup(termMon, *bindAddr)
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
					launched, ptListeners = proxy_transparent.ClientSetup(termMon, *target)
				}
			} else {
				log.Infof("%s - initializing server transport listeners", execName)
				if *bindAddr == "" {
					fmt.Println("%s - transparent mode requires a bindaddr", execName)
				} else {
					launched, ptListeners = proxy_transparent.ServerSetup(termMon, *bindAddr)
					fmt.Println("launched", launched, ptListeners)
				}
			}
		}
	} else {
		// Do the managed pluggable transport protocol configuration.
		log.Infof("%s - initializing PT 1.0 proxy", execName)
		if isClient {
			log.Infof("%s - initializing client transport listeners", execName)
			launched, ptListeners = proxy_socks5.ClientSetup(termMon)
		} else {
			log.Infof("%s - initializing server transport listeners", execName)
			launched, ptListeners = proxy_socks5.ServerSetup(termMon)
		}
	}

	if !launched {
		// Initialization failed, the client or server setup routines should
		// have logged, so just exit here.
		os.Exit(-1)
	}

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
