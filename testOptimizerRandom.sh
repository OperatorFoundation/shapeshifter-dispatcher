# This script runs a full end-to-end functional test of the dispatcher and the Replicant transport, using two netcat instances as the application server and application client.
# An alternative way to run this test is to run each command in its own terminal. Each netcat instance can be used to type content which should appear in the other.

# Update and build code
go get -u github.com/OperatorFoundation/shapeshifter-dispatcher

# Run a demo application server with netcat
nc -l 3333 &

# Run the transport server
./shapeshifter-dispatcher -transparent -server -state state -orport 127.0.0.1:3333 -transports shadow -bindaddr -127.0.0.1:2222 -optionsFile shadowServer.json -logLevel DEBUG -enableLogging &
./shapeshifter-dispatcher -transparent -server -state state -orport 127.0.0.1:3333 -transports obfs2 -bindaddr obfs2-127.0.0.1:2223 -logLevel DEBUG -enableLogging &
./shapeshifter-dispatcher -transparent -server -state state -orport 127.0.0.1:3333 -transports Replicant -bindaddr Replicant-127.0.0.1:2224 -optionsFile ReplicantServerConfig1.json -logLevel DEBUG -enableLogging &

sleep 5

# Run the transport client
./shapeshifter-dispatcher -transparent -client -state state -target 127.0.0.1:2222 -transports Optimizer -proxylistenaddr 127.0.0.1:1443 -optionsFile OptimizerRandom.json -logLevel DEBUG -enableLogging &

sleep 1

# Run a demo application client with netcat
echo "Test successful" | nc localhost 1443 &

sleep 5

echo "Testing complete. Killing processes."

killall shapeshifter-dispatcher
killall nc

echo "Done."
