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

If you want an end user that is trying to circumvent filtering on your network, or
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

The transports used by shapeshifter-dispatcher follow the Go Transport API in the [Pluggable Transports Specification v3.0](https://github.com/Pluggable-Transports/Pluggable-Transports-spec/blob/main/releases/PTSpecV3.0/Pluggable%20Transport%20Specification%20v3.0%20-%20Go%20Transport%20API%20v3.0.md).  
The dispatcher currently supports the following transports:
 * Replicant
 * Optimizer
 * shadow (Shadowsocks)
 
 Note: obs4 is no longer supported.  We recommend using Shadow in it's place.

#### Installation

The dispatcher is written in the Go programming language. To compile it you need
to install Go 1.17 or higher:

<https://golang.org/doc/install>

If you already have Go installed, make sure it is a compatible version:

    go version

The version should be 1.17 or higher.

If you get the error "go: command not found", then trying exiting your terminal
and starting a new one.

Get the git repository for shapeshifter-disptacher:

    git clone https://github.com/OperatorFoundation/shapeshifter-dispatcher.git

Go into that directory and build the command line executable:

    cd shapeshifter-dispatcher
    go install


This will fetch the source code for shapeshifter-dispatcher, and all the
dependencies, compile everything, and put the result in
<GOPATH>/bin/shapeshifter-dispatcher

You can find your GOPATH with:

    go env GOPATH


#### Running

Use either -client or -server to place the proxy into client or server mode,
respectively. Use -state to specify a directory to put transports state
information. Use -transports to specify which transports to launch.  Use -optionsFile to specify the directory where your config file is located

The default proxy mode is SOCKS5 (with optional PT 2.1 authentication protocol),
which can only proxy SOCKS5-aware TCP connections. For some transports, the
proxied connection will also need to know how to speak the PT 1.0 authentication
protocol. This requirement varies by the transport used.

Another TCP proxy mode is available, Transparent TCP, by using the -transparent
flag. In this mode, the proxy listens on a socket and any data from incoming
connections is forwarded over the transport.

UDP proxying can be enabled with the -udp flag. The default UDP mode is STUN
packet proxying. This requires that the application only send STUN packets, so
works for protocols such as WebRTC, which are based on top of STUN.

Another UDP proxy mode is available, Transparent UDP, by using the -transparent
flag with the -udp flag. In this mode, the proxy listens on a UDP socket and
any incoming packets are forwarded over the transport.

Only one proxy mode can be used at a time.

#### Running with Replicant

Replicant is Operator's flagship transport which can be tuned for each adversary.

Here are example command lines to run the dispatcher with the Replicant transport:

##### Server

For this example to work, you need an application server running. You can use netcat to run a simple server on port 3333:
 
    nc -l 3333

Now launch the transport server, telling it where to find the application server:

    <GOPATH>/bin/shapeshifter-dispatcher -transparent -server -state state -target 127.0.0.1:3333 -transports Replicant -bindaddr Replicant-127.0.0.1:2222 -optionsFile ConfigFiles/ReplicantServerConfigV3.json -logLevel DEBUG -enableLogging

This runs the server in transparent TCP proxy mode. The directory "state" is used
to hold transport state. The destination that the server will proxy to is
127.0.0.1, port 3333. The Replicant transport is enabled and bound to the address 127.0.0.1 and the port 2222. Logging is enabled and set to DEBUG level.
To access the Log for debugging purposes, look at state/dispatcher.log

To use Replicant, a config file is needed. A sample config file, located in ConfigFiles/ReplicantServerConfigV3.json, is provided purely for educational purposes and should not be used in actual production.

##### Client

    <GOPATH>/bin/shapeshifter-dispatcher -transparent -client -state state -transports Replicant -proxylistenaddr 127.0.0.1:1443 -optionsFile ConfigFiles/ReplicantClientConfigV3.json -logLevel DEBUG -enableLogging

This runs the client in transparent TCP proxy mode. The directory "state" is
used to hold transport state. The address of the server is specified as
127.0.0.1, port 2222. This is the same address as was specified on the server
command line above. For this demo to work, the dispatcher server needs to be
running on this host and port. The Replicant transport is enabled and bound to the
address 127.0.0.1 and the port 1443.

To use Replicant, a config file is needed. A sample config file, located in ConfigFiles/ReplicantClientConfigV3.json, is provided purely for educational purposes and should not be used in actual production.

Once the client is running, you can connect to the client address, which in this
case is 127.0.0.1, port 1443. For instance, you can telnet to this address:

    telnet 127.0.0.1 1443

Any bytes sent over this connection will be forwarded through the transport
server to the application server, which in the case of this demo is a netcat
server. You can also type bytes into the netcat server and they will appear
on the telnet client, once again being routed over the transport.

### Using Environment Variables

Using command line flags is convenient for testing. However, when launching the
dispatcher automatically from inside of an application, another option is to
use environment variables. Most of the functionality specified by command line
flags can also be set using environment variables instead.

### Running in SOCKS5 Mode

SOCKS5 mode is an older mode inherited from the PT1.0 specification and updated in PT2.0. Despite the name,
SOCKS5 mode does not provide a SOCKS proxy for use with SOCKS clients such as Firefox. Rather it uses the
SOCKS5 protocol as a way to communicate between a host application and Shapeshifter Dispatcher. The host application
must be aware of the special semantics used by this mode. While it is possible to configure Shapeshifter Dispatcher
to provide a traditional SOCKS proxy for use with SOCKS clients such as Firefox, that is not covered here.

SOCKS5 mode is not recommended for most users, use Transparent TCP mode instead.

Here are example command lines to run the dispatcher in SOCKS5 mode with the Replicant transport:

##### Server

For this example to work, you need an application server running. You can use netcat to run a simple server on port 3333:
 
    nc -l 3333

Now launch the transport server, telling it where to find the application server:

    <GOPATH>/bin/shapeshifter-dispatcher -server -state state -target 127.0.0.1:3333 -bindaddr shadow-127.0.0.1:2222 -transports shadow -optionsFile ConfigFiles/shadowServer.json -logLevel DEBUG -enableLogging

This runs the server in the default mode, which is SOCKS5 mode. The directory "state" is used
to hold transport state. The destination that the server will proxy to is 127.0.0.1, port 3333.
The Replicant transport is enabled and bound to the address 127.0.0.1 and the port
2222. Logging is enabled and set to DEBUG level. To access the Log for debugging purposes,
look at state/dispatcher.log

To use Replicant, a config file is needed. A sample config file, ReplicantServerConfigV3.json, is provided purely for educational purposes and should not be used in actual production.

##### Client

    <GOPATH>/bin/shapeshifter-dispatcher -client -state state -transports shadow -proxylistenaddr 127.0.0.1:1443 -optionsFile ConfigFiles/shadowClient.json -logLevel DEBUG -enableLogging

This runs the client in the default mode, which is SOCKS5 mode. The directory "state" is
used to hold transport state. The Replicant transport is enabled and bound to the
address 127.0.0.1 and the port 1443. Please note that you do not specify the server address with -target in SOCKS5
mode. This happens below, in the tsocks step.

To use Replicant, a config file is needed. A sample config file, ReplicantClientConfigV3.json, is provided purely for educational purposes and should not be used in actual production.

Once the client is running, you can connect to the client address, which in this
case is 127.0.0.1, port 1443. You will need to use a SOCKS5 client. Normally, this would be a host application
that you would write. For basic testing, you can install a tool such as tsocks.

For instance, on macOS, install tsocks:

    brew tap Anakros/homebrew-tsocks
    brew install --HEAD tsocks
    nano /usr/local/etc/tsocks.conf        
    
In your tsocks configuration file, add the following lines to tell it where to find the dispatcher client:

    server = 127.0.0.1
    server_port = 1443
    server_type = 5
    
It is important to check to make sure that your tsocks configuration is correct. If you have the wrong server
address or port, tsocks will connect you directly to the transport server and this will give confusing results.

Now you can use telnet to connect to the server and tsocks to route the traffic through SOCKS:

    tsocks telnet 127.0.0.1 2222

It is important to note that the address and port you telnet to is the address of the transport server. This
information is passed through the SOCKS5 protocol to the client by tsocks and it is how the client learns where
the server is located.

At this point, you should have a normal connection through the transport to the application server. Any bytes sent
over this connection will be forwarded through the transport server to the application server, which in the case of
this demo is a netcat server. You can also type bytes into the netcat server and they will appear
on the telnet client, once again being routed over the transport.

Please note that this is not an open SOCKS proxy that allows you to connect to any address on the Internet. You
can only connect to the application server associated with the transport server. The SOCKS protocol is only
used as a method of communication between a host application and the transport client. While we use tsocks as
the host application for this explanation, normally the host application would be a custom application provided by
you.

SOCKS5 mode is not recommended for most users, use Transparent TCP mode instead.

### Config generator

To generate a new pair of configs for any of the supported transports, run the following command:

    <GOPATH>/bin/shapeshifter-dispatcher -generateConfig -transport <transport name> -serverIP <serverIP:Port>

For Replicant, you can also add the flags -toneburst and/or -polish if you would like to enable the Starburst toneburst and the Darkstar polish respectively

### Credits

shapeshifter-dispatcher is descended from the Tor project's "obfs4proxy" tool.

 * David Fifield for goptlib
 * Adam Langley for the Go Elligator implementation.
 * Philipp Winter for the ScrambleSuit protocol.
 * Shadowsocks was developed by the Shadowsocks team. [whitepaper](https://shadowsocks.org/assets/whitepaper.pdf).
