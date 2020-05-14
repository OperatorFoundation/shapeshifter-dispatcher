# This script runs a full end-to-end functional test of the dispatcher and the Meek transport, using two netcat instances as the application server and application client.
# An alternative way to run this test is to run each command in its own terminal. Each netcat instance can be used to type content which should appear in the other.
FILENAME=testTCPMeekOutput.txt
# Update and build code
go get -u github.com/OperatorFoundation/shapeshifter-dispatcher

# remove text from the output file
rm $FILENAME

# Run a demo application server with netcat and write to the output file
nc -l 3333 >$FILENAME &

# Run the transport server
./shapeshifter-dispatcher -transparent -server -state state -orport 127.0.0.1:3333 -transports meekserver -bindaddr meekserver-127.0.0.1:2222 -optionsFile meek.json -logLevel DEBUG -enableLogging &

sleep 1

# Run the transport client
./shapeshifter-dispatcher -transparent -client -state state -target 127.0.0.1:2222 -transports meeklite -proxylistenaddr 127.0.0.1:1443 -optionsFile meek.json -logLevel DEBUG -enableLogging &

sleep 1

# Run a demo application client with netcat
echo "Test successful!" | nc localhost 1443 &

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
