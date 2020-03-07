# This script runs a full end-to-end functional test of the dispatcher and the Replicant transport, using two netcat instances as the application server and application client.
# An alternative way to run this test is to run each command in its own terminal. Each netcat instance can be used to type content which should appear in the other.

# Update and build code
go get -u github.com/OperatorFoundation/shapeshifter-dispatcher
go build

# Run a demo application server with netcat
nc -l 3333 &

# Run the transport server
export TOR_PT_SERVER_BINDADDR=Replicant-$1:2222
./shapeshifter-dispatcher -server -state state -orport 127.0.0.1:3333 -transports Replicant -optionsFile ReplicantServerConfig1.json -logLevel DEBUG -enableLogging &

sleep 1

# Run the transport client
./shapeshifter-dispatcher -client -state state -transports Replicant -proxylistenaddr 127.0.0.1:1443 -optionsFile ReplicantClientConfig1.json -logLevel DEBUG -enableLogging &

sleep 1

# Run a demo application client with tsocks and telnet
tsocks telnet $1 2222

echo "Testing complete. Killing processes."

killall shapeshifter-dispatcher
killall nc

echo "Done."

