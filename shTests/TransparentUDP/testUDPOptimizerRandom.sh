# This script runs a full end-to-end functional test of the dispatcher and the Replicant transportOptimizer transport with the Random Strategy. Each netcat instance can be used to type content which should appear in the other.
FILENAME=testUDPOptimizerRandomOutput.txt
# Update and build code
go get -u github.com/OperatorFoundation/shapeshifter-dispatcher

# remove text from the output file
rm $FILENAME

# Run a demo application server with netcat and write to the output file
nc -l -u 3333 >$FILENAME &

# Run the transport server
~/go/bin/shapeshifter-dispatcher -transparent -udp -server -state state -orport 127.0.0.1:3333 -transports shadow -bindaddr -127.0.0.1:2222 -optionsFile ../../ConfigFiles/shadowServerChaCha.json -logLevel DEBUG -enableLogging &
~/go/bin/shapeshifter-dispatcher -transparent -udp -server -state state -orport 127.0.0.1:3333 -transports obfs2 -bindaddr obfs2-127.0.0.1:2223 -logLevel DEBUG -enableLogging &
~/go/bin/shapeshifter-dispatcher -transparent -udp -server -state state -orport 127.0.0.1:3333 -transports Replicant -bindaddr Replicant-127.0.0.1:2224 -optionsFile ../../ConfigFiles/ReplicantServerConfigV2.json -logLevel DEBUG -enableLogging &

sleep 5

# Run the transport client
~/go/bin/shapeshifter-dispatcher -transparent -udp -client -state state -target 127.0.0.1:2222 -transports Optimizer -proxylistenaddr 127.0.0.1:1443 -optionsFile ../../ConfigFiles/OptimizerRandom.json -logLevel DEBUG -enableLogging &

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
