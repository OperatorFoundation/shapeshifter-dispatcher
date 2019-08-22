# The Operator Foundation

[Operator](https://operatorfoundation.org) makes useable tools to help people around the world with censorship, security, and privacy.

## Shapeshifter

The Shapeshifter project provides network protocol shapeshifting technology
(also sometimes referred to as obfuscation). The purpose of this technology is
to change the characteristics of network traffic so that it is not identified
and subsequently blocked by network filtering devices.

There are two components to Shapeshifter: transports and the dispatcher. Each
transport provide different approach to shapeshifting. These transports are
provided as a Go library which can be integrated directly into applications.
The dispatcher is a command line tool which provides a proxy that wraps the
transport library. It has several different proxy modes and can proxy both
TCP and UDP traffic.

If you are a tool developer working in the Go programming language, then you
probably want to use the transports library directly in your application.
<https://github.com/OperatorFoundation/shapeshifter-transports>

If you want a end user that is trying to circumvent filtering on your network or
you are a developer that wants to add pluggable transports to an existing tool
that is not written in the Go programming language, then you probably want the
dispatcher. Please note that familiarity with executing programs on the command
line is necessary to use this tool.
<https://github.com/OperatorFoundation/shapeshifter-dispatcher>

If you are looking for a complete, easy-to-use VPN that incorporates
shapeshifting technology and has a graphical user interface, consider
[Moonbounce](https://github.com/OperatorFoundation/Moonbounce), an application for macOS which incorporates shapeshifting without
the need to write code or use the command line.

### Shapeshifter Dispatcher

This is the repository for the shapeshifter-dispatcher command line proxy tool.
If you are looking for the transports is provides, they are here:
<https://github.com/OperatorFoundation/shapeshifter-transports>

The dispatcher implements the Pluggable Transports 2.1 draft 1 specification available here:
<https://github.com/Pluggable-Transports/Pluggable-Transports-spec/tree/master/releases/PTSpecV2.1Draft1>

The purpose of the dispatcher is to provide different proxy interfaces to using
transports. Through the use of these proxies, application traffic can be sent
over the network in a shapeshifted form that bypasses network filtering, allowing
the application to work on networks where it would otherwise be blocked or
heavily throttled.

The dispatcher currently supports the following proxy modes:
 * SOCKS5 (with optional PT 2.0 authentication protocol)
 * Transparent TCP
 * Transparent UDP
 * STUN UDP

The dispatcher currently supports the following transports:
 * obfs4
 * optimizer
 * shadow (Shadowsocks)

#### Installation

The dispatcher is written in the Go programming language. To compile it you need
to install Go 1.10.2 or higher:

<https://golang.org/doc/install>

If you just installed Go for the first time, you will need to create a directory
to keep all of your Go source code:

    mkdir ~/go

If you already have Go installed, make sure it is a compatible version:

    go version

The version should be 1.10.2 or higher.

If you get the error "go: command not found", then trying exiting your terminal
and starting a new one.

If you have a compatible Go installed, you should go to the directory where you
keep all of your Go source code and set your GOPATH:

    cd ~/go
    export GOPATH=~/go

Software written in Go is installed using the `go get` command:

    go get -u github.com/OperatorFoundation/shapeshifter-dispatcher/shapeshifter-dispatcher

This will fetch the source code for shapeshifter-dispatcher, and all the
dependencies, compile everything, and put the result in
bin/shapeshifter-dispatcher.

#### Running

Run without argument to get usage information:

    bin/shapeshifter-dispatcher

A minimal configuration requires at least --client, --state, and --transports.
Example:

    bin/shapeshifter-dispatcher --client --state state --transports obfs2

Use either --client or --server to place the proxy into client or server mode,
respectively. Use --state to specify a directory to put transports state
information. Use --transports to specify which transports to launch.

The default proxy mode is SOCKS5 (with optional PT 2.0 authentication protocol),
which can only proxy SOCKS5-aware TCP connections. For some transports, the
proxied connection will also need to know how to speak the PT 1.0 authentication
protocol. This requirement varies by the transport used.

Another TCP proxy mode is available, Transparent TCP, by using the --transparent
flag. In this mode, the proxy listens on a socket and any data from incoming
connections is forwarded over the transport.

UDP proxying can be enabled with the --udp flag. The default UDP mode is STUN
packet proxying. This requires that the application only send STUN packets, so
works for protocols such as WebRTC, which are based on top of STUN.

Another UDP proxy mode is available, Transparent UDP, by using the --transparent
flag with the --udp flag. In this mode, the proxy listens on a UDP socket and
any incoming packets are forwarded over the transport.

Only one proxy mode can be used at a time.

The full set of command line flags is specified in the Pluggable Transport 2.1
draft 1 specification.
<https://github.com/Pluggable-Transports/Pluggable-Transports-spec/tree/master/releases/PTSpecV2.1Draft1>

#### Running with obfs4

Here are example command lines to run the dispatcher with the obfs4 transport:

##### Server

    bin/shapeshifter-dispatcher -transparent -server -state state -orport 127.0.0.1:3333 -transports obfs4 -bindaddr obfs4-127.0.0.1:2222 -logLevel DEBUG -enableLogging -extorport 127.0.0.1:3334

This runs the server in transparent TCP proxy mode. The directory "state" is used
to hold transport state. The destination that the server will proxy to is
127.0.0.1, port 3333. For this demo to work, something needs to be running on
this host and port. You can use netcat to run a simple server with "nc -l 3333".
The obfs4 transport is enabled and bound to the address 127.0.0.1 and the port
2222. Logging is enabled and set to DEBUG level. The statistics reporting server
address is also required on the server and is set to 127.0.0.1, port 3334.
However, this service does not actually need to be running for the demo to work.
 To access this Log for debugging purposes, go to user/go/state/dispatcher.log

When the server is run for the first time, it will generate a new public key
and it will write it to a file in the state directory called
obfs4_bridgeline.txt. This information is needed by the dispatcher client. Look
in the file and retrieve the public key from the bridge line. It will look
similar to this:

    Bridge obfs4 <IP ADDRESS>:<PORT> <FINGERPRINT> cert=OfQAPDamjsRO90fDGlnZR5RNG659FZqUKUwxUHcaK7jIbERvNU8+EVF6rmdlvS69jVYrKw iat-mode=0

The cert parameter is what is needed for the dispatcher client.

##### Client

    bin/shapeshifter-dispatcher -transparent -client -state state -target 127.0.0.1:2222  -transports obfs4 -proxylistenaddr obfs4-127.0.0.1:1443 -options '{"cert": "OfQAPDamjsRO90fDGlnZR5RNG659FZqUKUwxUHcaK7jIbERvNU8+EVF6rmdlvS69jVYrKw", "iatMode": "0"}' -logLevel DEBUG -enableLogging

This runs the client in transparent TCP proxy mode. The directory "state" is
used to hold transport state. The address of the server is specified as
127.0.0.1, port 2222. This is the same address as was specified on the server
command line above. For this demo to work, the dispatcher server needs to be
running on this host and port. The obfs4 transport is enabled and bound to the
address 127.0.0.1 and the port 1443. The -options parameter is different for
every transport. For obfs4, the "cert" and "iatMode" parameters are required.
These can be found in the obfs4_bridgeline.txt in the server state directory,
which is generated by the server the first time that it is run. It is important
for the cert parameter to be correct, otherwise obfs4 will silently fail.
Logging is enabled and set to DEBUG level.

Once the client is running, you can connect to the client address, which in this
case is 127.0.0.1, port 1443. For instance, you can telnet to this address:

    telnet 127.0.0.1 1443

Any bytes sent over this connection will be forwarded through the transport
server to the application server, which in the case of this demo is a netcat
server. You can also type bytes into the netcat server and they will appear
on the telnet client, once again being routed over the transport.

##### Environment Variables

Using command line flags is convenient for testing. However, when launching the
dispatcher automatically from inside of an application, another option is to
use environment variables. Most of the functionality specified by command line
flags can also be set using environment variables instead.

The full set of environment variables is specified in the Pluggable Transport
2.1 draft 1 specification.
<https://github.com/Pluggable-Transports/Pluggable-Transports-spec/tree/master/releases/PTSpecV2.1Draft1>

### Credits

shapeshifter-dispatcher is based on the Tor project's "obfs4proxy" tool.

 * Yawning Angel for obfs4proxy
 * David Fifield for goptlib
 * Adam Langley for the Go Elligator implementation.
 * Philipp Winter for the ScrambleSuit protocol.
