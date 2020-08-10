# This script runs a full end-to-end functional test of the dispatcher and the Optimizer transport with the Rotate Strategy, using two netcat instances as the application server and application client.
# An alternative way to run this test is to run each command in its own terminal. Each netcat instance can be used to type content which should appear in the other.
FILENAME=testSocksTCPOptimizerRotateOutput.txt
# Update and build code
go build

# remove text from the output file
rm $FILENAME

# Run a demo application server with netcat and write to the output file
nc -l 3333 >$FILENAME &

# Run the transport server
./shapeshifter-dispatcher -server -state state -target 127.0.0.1:3333 -bindaddr 127.0.0.1:2222 -transports shadow -optionsFile shadowServer.json -logLevel DEBUG -enableLogging &
./shapeshifter-dispatcher -server -state state -target 127.0.0.1:3333 -bindaddr 127.0.0.1:2222 -transports obfs2 -logLevel DEBUG -enableLogging &
./shapeshifter-dispatcher -server -state state -target 127.0.0.1:3333 -bindaddr 127.0.0.1:2222 -transports Replicant -optionsFile ReplicantServerConfigV3.json -logLevel DEBUG -enableLogging &

sleep 5

# Run the transport client
./shapeshifter-dispatcher -client -state state -transports Optimizer -proxylistenaddr 127.0.0.1:1443 -optionsFile OptimizerRotate.json -logLevel DEBUG -enableLogging &

sleep 1

# Run a demo application client with netcat
go test -run SocksTCPOptimizerRotate

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
