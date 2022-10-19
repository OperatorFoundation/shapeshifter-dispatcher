#!/bin/bash
# This script runs a full end-to-end functional test of the dispatcher and the Replicant transportOptimizer transport with the Random Strategy. Each netcat instance can be used to type content which should appear in the other.
FILENAME=testStunUDPOptimizerRandomOutput.txt

if [[ -z "${GOPATH}" ]]; then
  echo "your GOPATH variable is not set. Temporarily setting to HOME/go"
fi
GOPATH=${GOPATH:-"$HOME/go"}

# Update and build code
go install

# remove text from the output file
rm shTests/StunUDP/$FILENAME

# Run a demo application server with netcat and write to the output file
nc -l 3333 >shTests/StunUDP/$FILENAME &

# Run the transport server
"$GOPATH"/bin/shapeshifter-dispatcher -udp -server -state state -target 127.0.0.1:3333 -transports shadow -bindaddr shadow-127.0.0.1:2222 -optionsFile ../../ConfigFiles/shadowServer.json -logLevel DEBUG -enableLogging &
"$GOPATH"/bin/shapeshifter-dispatcher -udp -server -state state -target 127.0.0.1:3333 -transports Starbridge -bindaddr Starbridge-127.0.0.1:2223 -optionsFile ../../ConfigFiles/StarbridgeServerConfig.json -logLevel DEBUG -enableLogging &
"$GOPATH"/bin/shapeshifter-dispatcher -udp -server -state state -target 127.0.0.1:3333 -transports Replicant -bindaddr Replicant-127.0.0.1:2224 -optionsFile ../../ConfigFiles/ReplicantServerConfigV3.json -logLevel DEBUG -enableLogging &

sleep 5

# Run the transport client
"$GOPATH"/bin/shapeshifter-dispatcher -udp -client -state state -transports Optimizer -proxylistenaddr 127.0.0.1:1443 -optionsFile ../../ConfigFiles/OptimizerRandom.json -logLevel DEBUG -enableLogging &

sleep 1

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
