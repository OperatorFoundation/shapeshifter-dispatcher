#!/bin/bash
# This script runs a full end-to-end functional test of the dispatcher and the Replicant transport, using two netcat instances as the application server and application client.
# An alternative way to run this test is to run each command in its own terminal. Each netcat instance can be used to type content which should appear in the other.
FILENAME=testTCPReplicantOutput.txt

# Update and build code
go install

# remove text from the output file
rm $FILENAME

# Run a demo application server with netcat and write to the output file
nc -l 3333 >$FILENAME &

# Run the transport server
shapeshifter-dispatcher -transparent -server -state state -target 127.0.0.1:3333 -transports Replicant -bindaddr Replicant-127.0.0.1:2222 -optionsFile ../../ConfigFiles/ReplicantServerConfigV3.json -logLevel DEBUG -enableLogging &

sleep 1

# Run the transport client

shapeshifter-dispatcher -transparent -client -state state -transports Replicant -proxylistenaddr 127.0.0.1:1443 -optionsFile ../../ConfigFiles/ReplicantClientConfigV3.json -logLevel DEBUG -enableLogging &

sleep 1

# Run a demo application client with netcat
pushd shTests/TransparentTCP
go test -run TransparentTCP
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

