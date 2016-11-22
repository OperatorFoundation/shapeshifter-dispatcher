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
Moonbounce, an application for macOS which incorporates shapeshifting without
the need to write code or use the command line.

### Shapeshifter Dispatcher

This is the repository for the shapeshifter-dispatcher command line proxy tool.
If you are looking for the transports is provides, they are here:
<https://github.com/OperatorFoundation/shapeshifter-transports>

The dispatcher implements the Pluggable Transports 2.0 draft 1 specification available here:
<http://www.pluggabletransports.info/assets/PTSpecV2Draft1.pdf>

The purpose of the dispatcher is to provide different proxy interfaces to using
transports. Through the use of these proxies, application traffic can be sent
over the network in a shapeshifted form that bypasses network filtering, allowing
the application to work on networks where it would otherwise be blocked or
heavily throttled.

The dispatcher currently supports the following proxy modes:
 * SOCKS5 (with optional PT 1.0 authentication protocol)
 * Transparent TCP
 * Transparent UDP
 * STUN UDP

The dispatcher currently supports the following transports:
 * meek
 * obfs4
 * obfs3
 * obfs2
 * scramblesuit

#### Installation

The dispatcher is written in the Go programming language. To compile it you need
to install Go:

<https://golang.org/doc/install>

If you just installed Go for the first time, you will need to create a directory
to keep all of your Go source code:

    mkdir ~/go
    export GOPATH=~/go
    cd ~/go

Software written in Go is installed using the `go get` command:

    go get github.com/OperatorFoundation/shapeshifter-dispatcher

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

The default proxy mode is SOCKS5 (with optional PT 1.0 authentication protocol),
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

The full set of command line flags is specified in the Pluggable Transport 2.0
draft 1 specification.
<http://www.pluggabletransports.info/assets/PTSpecV2Draft1.pdf>

##### Environment Variables

Using command line flags is convenient for testing. However, when launching the
dispatcher automatically from inside of an application, another option is to
use environment variables. Most of the functionality specified by command line
flags can also be set using environment variables instead.

The full set of environment variables is specified in the Pluggable Transport
2.0 draft 1 specification.
<http://www.pluggabletransports.info/assets/PTSpecV2Draft1.pdf>

### Credits

shapeshifter-dispatcher is based on the Tor project's "obfs4proxy" tool.

 * Yawning Angel for obfs4proxy
 * David Fifield for goptlib
 * Adam Langley for the Go Elligator implementation.
 * Philipp Winter for the ScrambleSuit protocol.
