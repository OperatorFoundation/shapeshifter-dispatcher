# This script runs a full end-to-end functional test of the dispatcher and the Optimizer transport with the MinimizeDialDuration Strategy, using two netcat instances as the application server and application client.
# An alternative way to run this test is to run each command in its own terminal. Each netcat instance can be used to type content which should appear in the other.
FILENAME=testUDPOptimizerMinimizeDialDurationOutput.txt
# Update and build code
go build

# remove text from the output file
rm $FILENAME

# Run a demo application server with netcat and write to the output file
nc -l -u 3333 >$FILENAME &

# Run the transport server
./shapeshifter-dispatcher -transparent -udp -server -state state -orport 127.0.0.1:3333 -transports shadow -bindaddr -127.0.0.1:2222 -optionsFile shadowServer.json -logLevel DEBUG -enableLogging &
./shapeshifter-dispatcher -transparent -udp -server -state state -orport 127.0.0.1:3333 -transports obfs2 -bindaddr obfs2-127.0.0.1:2223 -logLevel DEBUG -enableLogging &
./shapeshifter-dispatcher -transparent -udp -server -state state -orport 127.0.0.1:3333 -transports Replicant -bindaddr Replicant-127.0.0.1:2224 -optionsFile ReplicantServerConfigV2.json -logLevel DEBUG -enableLogging &

sleep 5

# Run the transport client
./shapeshifter-dispatcher -transparent -udp -client -state state -transports Optimizer -proxylistenaddr 127.0.0.1:1443 -optionsFile OptimizerMinimizeDialDuration.json -logLevel DEBUG -enableLogging &

sleep 1

# Run a demo application client with netcat
go test -run TransparentUDP

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
