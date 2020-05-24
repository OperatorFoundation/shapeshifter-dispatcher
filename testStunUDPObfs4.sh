# This script runs a full end-to-end functional test of the dispatcher and the Obfs4 transport, using two netcat instances as the application server and application client.
# An alternative way to run this test is to run each command in its own terminal. Each netcat instance can be used to type content which should appear in the other.
FILENAME=testStunUDPObfs4Output.txt
OS=$(uname)
# Update and build code
go get -u github.com/OperatorFoundation/shapeshifter-dispatcher

# remove text from the output file
rm $FILENAME

# Run a demo application server with netcat and write to the output file
nc -l -u 3333 >$FILENAME &

if [ "$OS" = "Darwin" ]
then
  STATEPATH=$HOME/shapeshifter-dispatcher/stateDir
else
  STATEPATH=$HOME/gopath/src/github.com/OperatorFoundation/shapeshifter-dispatcher/stateDir
fi

# Run the transport server
./shapeshifter-dispatcher -udp -server -state "$STATEPATH" -orport 127.0.0.1:3333 -transports obfs4 -bindaddr obfs4-127.0.0.1:2222 -logLevel DEBUG -enableLogging &

sleep 1

CERTSTRING=$(cat "$STATEPATH/obfs4_bridgeline.txt" | grep cert | awk '{print $6}')
CERT=${CERTSTRING:5}
echo "$STATEPATH"
echo "$CERT"
echo "$OS"
echo "{\"cert\": \"$CERT\", \"iat-mode\": \"0\"}" >obfs4.json

# Run the transport client
./shapeshifter-dispatcher -udp -client -state "$STATEPATH" -target 127.0.0.1:2222 -transports obfs4 -proxylistenaddr 127.0.0.1:1443 -optionsFile obfs4.json -logLevel DEBUG -enableLogging &

sleep 5

# Run a demo application client with netcat
go test -run StunUDP

sleep 1

OS=$(uname)

if [ "$OS" = "Darwin" ]
then
  FILESIZE=$(stat -f%z "$FILENAME")
else
  FILESIZE=$(stat -c%s "$FILENAME")
fi

if [ "$FILESIZE" = "0" ]
then
  echo "Test Failed"
  killall shapeshifter-dispatcher
  killall nc
  exit 1
fi

echo "Testing complete. Killing processes."

killall shapeshifter-dispatcher
killall nc

echo "Done."
