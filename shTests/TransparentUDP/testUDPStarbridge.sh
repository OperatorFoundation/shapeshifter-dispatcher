#!/bin/bash
# This script runs a full end-to-end functional test of the dispatcher and the Replicant transport, using two netcat instances as the application server and application client.
# An alternative way to run this test is to run each command in its own terminal. Each netcat instance can be used to type content which should appear in the other.
FILENAME=testUDPStarbridgeOutput.txt

if [[ -z "${GOPATH}" ]]; then
  echo "your GOPATH variable is not set. Temporarily setting to HOME/go"
fi
GOPATH=${GOPATH:-"$HOME/go"}

# Update and build code
go install

# remove text from the output file
rm shTests/TransparentUDP/$FILENAME

# Run a demo application server with netcat and write to the output file
nc -l 3333 >shTests/TransparentUDP/$FILENAME &

# Run the transport server
"$GOPATH"/bin/shapeshifter-dispatcher -transparent -udp -server -state state -target 127.0.0.1:3333 -transports Starbridge -bindaddr Starbridge-127.0.0.1:2222 -optionsFile ../../ConfigFiles/StarbridgeServerConfig.json -logLevel DEBUG -enableLogging &

sleep 1

# Run the transport client
"$GOPATH"/bin/shapeshifter-dispatcher -transparent -udp -client -state state -transports Starbridge -proxylistenaddr 127.0.0.1:1443 -optionsFile ../../ConfigFiles/StarbridgeClientConfig.json -logLevel DEBUG -enableLogging &
sleep 1

# Run a demo application client with netcat
pushd shTests/TransparentUDP
go test -run TransparentUDP
popd

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

