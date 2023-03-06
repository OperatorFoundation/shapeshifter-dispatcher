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
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/OperatorFoundation/shapeshifter-dispatcher/common/pt_extras"
	"github.com/OperatorFoundation/shapeshifter-dispatcher/transports"
	"github.com/kataras/golog"

	"github.com/OperatorFoundation/shapeshifter-dispatcher/modes/pt_socks5"
	"github.com/OperatorFoundation/shapeshifter-dispatcher/modes/stun_udp"
	"github.com/OperatorFoundation/shapeshifter-dispatcher/modes/transparent_tcp"
	"github.com/OperatorFoundation/shapeshifter-dispatcher/modes/transparent_udp"
)

const (
	dispatcherVersion = "0.0.7-dev"
	dispatcherLogFile = "dispatcher.log"
)

var stateDir string

func getVersion() string {
	return fmt.Sprintf("dispatcher-%s", dispatcherVersion)
}

const (
	socks5 = iota
	transparentTCP
	transparentUDP
	stunUDP
)

func main() {

	// Handle the command line arguments.
	_, execName := path.Split(os.Args[0])

	flag.Usage = func() {
		_, _ = fmt.Fprintf(os.Stderr, "shapeshifter-dispatcher is a PT v3.0 proxy supporting multiple transports and proxy modes\n\n")
		_, _ = fmt.Fprintf(os.Stderr, "Usage:\n\t%s -client -state [statedir] -version 2 -transports [transport1,transport2,...]\n\n", os.Args[0])
		_, _ = fmt.Fprintf(os.Stderr, "Example:\n\t%s -client -state state -version 2 -transports obfs2\n\n", os.Args[0])
		_, _ = fmt.Fprintf(os.Stderr, "Flags:\n\n")
		flag.PrintDefaults()
	}

	// Parsing flags starts here, but variables are not set to actual values until flag.Parse() is called.
	// PT 2.1 specification, 3.3.1.1. Common Configuration Parameters
	var ptversion = flag.String("ptversion", "2.1", "Specify the Pluggable Transport protocol version to use")

	statePath := flag.String("state", "state", "Specify the directory to use to store state information required by the transports")
	exitOnStdinClose := flag.Bool("exit-on-stdin-close", false, "Set to true to force the dispatcher to close when the stdin pipe is closed")

	transportsList := flag.String("transports", "", "Specify transports to enable")

	//This is for proposal no.9
	transport := flag.String("transport", "", "Specify a single transport to enable")
	//copy old code
	serverBindPort := flag.String("bindport", "", "Specify the bind address port for transparent server")
	serverBindHost := flag.String("bindhost", "", "Specify the bind address host for transparent server")
	targetHost := flag.String("targethost", "", "Specify transport server destination address port")
	targetPort := flag.String("targetport", "", "Specify transport server destination address host")
	proxyListenHost := flag.String("proxylistenhost", "", "Specify the bind address for the local SOCKS server host provided by the client")
	proxyListenPort := flag.String("proxylistenport", "", "Specify the bind address for the local SOCKS server port provided by the client")
	modeName := flag.String("mode", "", "Specify which mode is being used: transparent-TCP, transparent-UDP, socks5, or STUN")

	// PT 2.1 specification, 3.3.1.2. Pluggable PT Client Configuration Parameters
	proxy := flag.String("proxy", "", "Specify an HTTP or SOCKS4a proxy that the PT needs to use to reach the Internet")

	// PT 2.1 specification, 3.3.1.3. Pluggable PT Server Environment Variables
	options := flag.String("options", "", "Specify the transport options for the server")
	if *options != "" {
		println("--> -options flag found: ", *options)
	}

	bindAddr := flag.String("bindaddr", "", "Specify the bind address for transparent server")
	extorport := flag.String("extorport", "", "Specify the address of a server implementing the Extended OR Port protocol, which is used for per-connection metadata")
	authcookie := flag.String("authcookie", "", "Specify an authentication cookie, for use in authenticating with the Extended OR Port")

	// Experimental flags under consideration for PT 2.1
	socksAddr := flag.String("proxylistenaddr", "", "Specify the bind address for the local SOCKS server provided by the client")
	optionsFile := flag.String("optionsFile", "", "store all the options in a single file")

	// Additional command line flags inherited from obfs4proxy
	showVer := flag.Bool("showVersion", false, "Print version and exit")
	logLevelStr := flag.String("logLevel", "ERROR", "Log level (ERROR/WARN/INFO/DEBUG)")
	enableLogging := flag.Bool("enableLogging", false, "Log to [state]/"+dispatcherLogFile)
	ipcLogLevelStr := flag.String("ipcLogLevel", "NONE", "IPC Log level (ERROR/WARN/INFO/DEBUG/NONE)")

	// Flags for config generation
	generateConfig := flag.Bool("generateConfig", false, "Generate a config for the specified transport")
	serverAddress := flag.String("serverIP", "", "Specify the IP address of the server to use in the config")
	toneburst := flag.Bool("toneburst", false, "Use the starburst toneburst for the Replicant config generation")
	polish := flag.Bool("polish", false, "Use the Darkstar polish for the Replicant config generation")

	// Additional command line flags added to shapeshifter-dispatcher
	clientMode := flag.Bool("client", false, "Enable client mode")
	serverMode := flag.Bool("server", false, "Enable server mode")
	transparent := flag.Bool("transparent", false, "Enable transparent proxy mode. The default is protocol-aware proxy mode (socks5 for TCP, STUN for UDP)")
	udp := flag.Bool("udp", false, "Enable UDP proxy mode. The default is TCP proxy mode.")
	target := flag.String("target", "", "Specify transport server destination address")
	enableLocket := flag.Bool("enableLocket", false, "Log to [state]/"+dispatcherLogFile+" using Locket")
	flag.Parse() // Flag variables are set to actual values here.

	// Start validation of command line arguments

	if *generateConfig {
		switch strings.ToLower(*transport) {
			case "shadow":
				transports.CreateShadowConfigs(*serverAddress)
			case "starbridge":
				transports.CreateStarbridgeConfigs(*serverAddress)
			case "replicant":
				transports.CreateReplicantConfigs(*serverAddress, *toneburst, *polish)
			default:
				// FIXME: add print/log in case of wrong name
				return
		}
	} 

	if *showVer {
		fmt.Printf("%s\n", getVersion())
		os.Exit(0)
	}

	logPath := path.Join(stateDir, dispatcherLogFile)
	logFile, logFileErr := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if logFileErr != nil {
		println("could open logFile")
	}

	golog.SetOutput(logFile)

	if enableLogging != nil {
		lowerCaseLogLevelStr := strings.ToLower(*logLevelStr)
		golog.SetLevel(lowerCaseLogLevelStr)
	} else {
		golog.SetLevel("fatal")
	}

	ipcLogLevel, ipcLogLevelError := validateIPCLogLevel(*ipcLogLevelStr)
	if ipcLogLevelError != nil {
		println(ipcLogLevel)
		golog.Errorf("could not validate IPC log level %s", ipcLogLevelError)
		return
	}

	// Determine if this is a client or server, initialize the common state.
	launched := false
	isClient, err := checkIsClient(*clientMode, *serverMode)
	if err != nil {
		flag.Usage()
		golog.Fatalf("[ERROR]: %s - either --client or --server is required, or configure using PT 2.0 environment variables", execName)
	}
	if stateDir, err = makeStateDir(*statePath); err != nil {
		flag.Usage()
		golog.Fatalf("[ERROR]: %s - No state directory: Use --state", execName)
	}
	if *options != "" && *optionsFile != "" {
		golog.Fatal("You should not specify -options and -optionsFile at the same time.")
	}
	if *optionsFile != "" {
		_, err := os.Stat(*optionsFile)
		if err != nil {
			golog.Errorf("Received an error while attempting to parse the options file %s: %s", *optionsFile, err.Error())
		} else {
			contents, readErr := ioutil.ReadFile(*optionsFile)
			if readErr != nil {
				golog.Errorf("Failed to open the optionsFile: %s", *optionsFile)
			} else {
				*options = string(contents)
			}
		}
	}

	transportValidationError := validateTransports(transport, transportsList)
	if transportValidationError != nil {
		golog.Errorf("Failed to validate transports: %s", transportValidationError)
		return
	}

	if *transport != "" && *transportsList == "" {
		transportsList = transport
	}

	modeValidationError := validateMode(modeName, transparent, udp)
	if modeValidationError != nil {
		golog.Errorf("Failed to validate the mode: %s", modeValidationError)
		return
	}

	mode, modeError := determineMode(*modeName, *transparent, *udp)
	if modeError != nil {
		golog.Errorf("Invalid mode name: %s", *modeName)
		return
	}

	if isClient {
		proxyListenValidationError := validateProxyListenAddr(proxyListenHost, proxyListenPort, socksAddr)
		if proxyListenValidationError != nil {
			golog.Errorf("could not validate: %s", proxyListenValidationError)
			golog.Infof("proxylistenhost: %s", *proxyListenHost)
			golog.Infof("proxylistenport: %s", *proxyListenPort)
			golog.Infof("proxylistenaddr: %s", *socksAddr)
			return
		}

		if *proxyListenHost != "" && *proxyListenPort != "" && *socksAddr == "" {
			newSocksAddr := *proxyListenHost + ":" + *proxyListenPort
			socksAddr = &newSocksAddr
		}

		if *socksAddr == "" {
			*socksAddr = "127.0.0.1:0"
		}

		if mode == socks5 {
			targetValidationError := validatetargetSocks5(targetHost, targetPort, target)
			if targetValidationError != nil {
				golog.Errorf("could not validate: %s", targetValidationError)
				return
			}

		} else {
			targetValidationError := validatetarget(isClient, targetHost, targetPort, target)
			if targetValidationError != nil {
				println("could not validate: ", targetValidationError.Error())
				golog.Errorf("could not validate: %s", targetValidationError)
				return
			}
			if *targetHost != "" && *targetPort != "" && *target == "" {
				newTarget := *targetHost + ":" + *targetPort
				bindAddr = &newTarget
			}
		}

	} else {
		if mode == socks5 {
			serverBindValidationError := validateSocksServerBindAddr(serverBindHost, serverBindPort, bindAddr)
			if serverBindValidationError != nil {
				golog.Errorf("could not validate: %s", serverBindValidationError)
				return
			}
		} else {
			serverBindValidationError := validateServerBindAddr(transport, serverBindHost, serverBindPort, bindAddr)
			if serverBindValidationError != nil {
				golog.Errorf("could not validate: %s", serverBindValidationError)
				return
			}

			if *transport != "" && *serverBindHost != "" && *serverBindPort != "" && *bindAddr == "" {
				newBindAddr := *transport + "-" + *serverBindHost + ":" + *serverBindPort
				bindAddr = &newBindAddr
			}
		}
	}
	// Finished validation of command line arguments

	golog.Infof("%s - launched", getVersion())

	if isClient {
		golog.Infof("%s - initializing client transport listeners", execName)

		switch mode {
		case socks5:
			golog.Infof("%s - initializing client transport listeners", execName)
			ptClientProxy, names, nameErr := getClientNames(ptversion, transportsList, proxy)
			if nameErr != nil {
				golog.Errorf("must specify -version and -transports")
				return
			}
			launched = pt_socks5.ClientSetup(*socksAddr, ptClientProxy, names, *options, *enableLocket, stateDir)
		case transparentTCP:
			ptClientProxy, names, nameErr := getClientNames(ptversion, transportsList, proxy)
			if nameErr != nil {
				golog.Errorf("must specify -version and -transports")
				return
			}
			launched = transparent_tcp.ClientSetup(*socksAddr, ptClientProxy, names, *options, *enableLocket, stateDir)
		case transparentUDP:
			ptClientProxy, names, nameErr := getClientNames(ptversion, transportsList, proxy)
			if nameErr != nil {
				golog.Errorf("must specify -version and -transports")
				return
			}
			launched = transparent_udp.ClientSetup(*socksAddr, ptClientProxy, names, *options)
		case stunUDP:
			ptClientProxy, names, nameErr := getClientNames(ptversion, transportsList, proxy)
			if nameErr != nil {
				golog.Errorf("must specify -version and -transports")
				return
			}
			launched = stun_udp.ClientSetup(*socksAddr, ptClientProxy, names, *options)
		default:
			golog.Errorf("unsupported mode %d", mode)
		}
	} else {
		golog.Infof("initializing server transport listeners")

		switch mode {
		case socks5:
			golog.Infof("%s - initializing socks5 server transport listeners", execName)
			ptServerInfo := getServerInfo(bindAddr, options, transportsList, target, extorport, authcookie)
			launched = pt_socks5.ServerSetup(ptServerInfo, stateDir, *options, *enableLocket)
		case transparentTCP:
			golog.Infof("%s - initializing transparentTCP server transport listeners", execName)
			ptServerInfo := getServerInfo(bindAddr, options, transportsList, target, extorport, authcookie)
			launched = transparent_tcp.ServerSetup(ptServerInfo, stateDir, *options, *enableLocket)
		case transparentUDP:
			// launched = transparent_udp.ServerSetup(termMon, *bindAddr, *target)
			ptServerInfo := getServerInfo(bindAddr, options, transportsList, target, extorport, authcookie)
			launched = transparent_udp.ServerSetup(ptServerInfo, stateDir, *options)
		case stunUDP:
			ptServerInfo := getServerInfo(bindAddr, options, transportsList, target, extorport, authcookie)
			launched = stun_udp.ServerSetup(ptServerInfo, stateDir, *options)
		default:
			golog.Errorf("unsupported mode %d", mode)
		}
	}

	if !launched {
		// Initialization failed, the client or server setup routines should
		// have logged, so just exit here.
		println("no pluggable transports were launched")
		os.Exit(-1)
	}

	golog.Infof("%s - accepting connections", execName)

	if *exitOnStdinClose {
		_, _ = io.Copy(ioutil.Discard, os.Stdin)
		os.Exit(-1)
	} else {
		select {}
	}
}

func determineMode(mode string, isTransparent bool, isUDP bool) (int, error) {
	if mode != "" {
		switch mode {
		case "socks5":
			return socks5, nil
		case "transparent-TCP":
			return transparentTCP, nil
		case "transparent-UDP":
			return transparentUDP, nil
		case "STUN":
			return stunUDP, nil
		default:
			return -1, errors.New("invalid mode")
		}
	}
	if isTransparent && isUDP {
		golog.Infof("initializing transparent proxy")
		golog.Infof("initializing UDP transparent proxy")
		return transparentUDP, nil
	} else if isTransparent {
		golog.Infof("initializing transparent proxy")
		golog.Infof("initializing TCP transparent proxy")
		return transparentTCP, nil
	} else if isUDP {
		golog.Infof("initializing STUN UDP proxy")
		return stunUDP, nil
	} else {
		golog.Infof("initializing PT 2.1 socks5 proxy")
		return socks5, nil
	}
}

func checkIsClient(client bool, server bool) (bool, error) {
	if client {
		return true, nil
	} else if server {
		return false, nil
	} else {
		return true, nil
	}
}

func makeStateDir(statePath string) (string, error) {
	if statePath != "" {
		err := os.MkdirAll(statePath, 0700)
		return statePath, err
	}

	return statePath, nil
}

func getClientNames(ptversion *string, transportsList *string, proxy *string) (clientProxy *url.URL, names []string, retErr error) {
	var ptClientInfo pt_extras.ClientInfo

	if *transportsList == "*" {
		ptClientInfo = pt_extras.ClientInfo{MethodNames: transports.Transports()}
	} else {
		ptClientInfo = pt_extras.ClientInfo{MethodNames: strings.Split(*transportsList, ",")}
	}

	ptClientProxy, proxyErr := pt_extras.PtGetProxy(proxy)
	if proxyErr != nil {
		return nil, nil, proxyErr
	} else if ptClientProxy != nil {
		pt_extras.PtProxyDone()
	}

	return ptClientProxy, ptClientInfo.MethodNames, nil
}

func getServerInfo(bindaddrList *string, options *string, transportList *string, target *string, extorport *string, authcookie *string) pt_extras.ServerInfo {
	var ptServerInfo pt_extras.ServerInfo
	var err error
	var bindaddrs []pt_extras.Bindaddr

	bindaddrs, err = getServerBindaddrs(bindaddrList, options, transportList)
	if err != nil {
		golog.Errorf(err.Error())
		golog.Errorf("Error parsing bindaddrs %q %q %q", *bindaddrList, *options, *transportList)
		return ptServerInfo
	}

	ptServerInfo = pt_extras.ServerInfo{Bindaddrs: bindaddrs}
	ptServerInfo.OrAddr, err = pt_extras.ResolveAddr(*target)
	if err != nil {
		golog.Errorf("Error resolving OR address %q %q", *target, err)
		return ptServerInfo
	}

	if *authcookie != "" {
		ptServerInfo.AuthCookiePath = *authcookie
	}

	if *extorport != "" {
		ptServerInfo.ExtendedOrAddr, err = pt_extras.ResolveAddr(*extorport)
		if err != nil {
			golog.Errorf("Error resolving Extended OR address %q %q", *extorport, err)
			return ptServerInfo
		}
	}

	return ptServerInfo
}

// Return an array of Bindaddrs.
func getServerBindaddrs(bindaddrList *string, options *string, transports *string) ([]pt_extras.Bindaddr, error) {
	var result []pt_extras.Bindaddr
	var serverBindaddr string
	var serverTransports string

	// Get the list of all requested bindaddrs.
	if *bindaddrList != "" {
		serverBindaddr = *bindaddrList
	}

	for _, spec := range strings.Split(serverBindaddr, ",") {
		var bindaddr pt_extras.Bindaddr

		parts := strings.SplitN(spec, "-", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("-bindaddr: %q: doesn't contain \"-\"", spec)
		}
		bindaddr.MethodName = parts[0]
		addr, err := pt_extras.ResolveAddr(parts[1])
		if err != nil {
			return nil, fmt.Errorf("-bindaddr: %q: %s", spec, err.Error())
		}
		bindaddr.Addr = addr
		bindaddr.Options = *options
		result = append(result, bindaddr)
	}

	if transports == nil {
		return nil, errors.New("must specify -transport or -transports in server mode")
	} else {
		serverTransports = *transports
	}
	result = pt_extras.FilterBindaddrs(result, strings.Split(serverTransports, ","))
	if len(result) == 0 {
		golog.Errorf("no valid bindaddrs")
	}
	return result, nil
}
