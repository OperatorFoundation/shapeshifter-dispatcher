# This script runs a full end-to-end functional test of the dispatcher and the Shadow transport, using two netcat instances as the application server and application client.
# An alternative way to run this test is to run each command in its own terminal. Each netcat instance can be used to type content which should appear in the other.
# Update and build code
go build

# Test -transport
echo "* transport"

./shapeshifter-dispatcher -transparent -server -state state -target 127.0.0.1:3333 -transport shadow -bindaddr shadow-127.0.0.1:2222 -optionsFile shadowServer.json -logLevel DEBUG -enableLogging &

sleep 1

killall shapeshifter-dispatcher

#Test -bindhost, -bindport
echo "* bindhost and bindport"
./shapeshifter-dispatcher -transparent -server -state state -target 127.0.0.1:3333 -transport shadow -bindhost 127.0.0.1 -bindport 2222 -optionsFile shadowServer.json -logLevel DEBUG -enableLogging &

sleep 1

killall shapeshifter-dispatcher

# Test -targethost, targetport
echo "* targethost and targetport"
./shapeshifter-dispatcher -transparent -client -state state -transport shadow -proxylistenaddr 127.0.0.1:1443 -optionsFile shadowClient.json -logLevel DEBUG -enableLogging &

sleep 1

killall shapeshifter-dispatcher

# Test -proxylistenhost, -proxylistenport
echo "* proxylistenhost and proxylistenport"

./shapeshifter-dispatcher -transparent -client -state state -transport shadow -proxylistenhost 127.0.0.1 -proxylistenport 1443 -optionsFile shadowClient.json -logLevel DEBUG -enableLogging &

sleep 1

killall shapeshifter-dispatcher

# Test -mode transparent-TCP
echo "* TransparentTCP"

./shapeshifter-dispatcher -mode transparent-TCP -client -state state -transport shadow -proxylistenaddr 127.0.0.1:1443 -optionsFile shadowClient.json -logLevel DEBUG -enableLogging &

sleep 1

killall shapeshifter-dispatcher

# Test -mode transparent-UDP
echo "* TransparentUDP"

./shapeshifter-dispatcher -mode transparent-UDP -client -state state -transport shadow -proxylistenaddr 127.0.0.1:1443 -optionsFile shadowClient.json -logLevel DEBUG -enableLogging &

sleep 1

killall shapeshifter-dispatcher

# Test -mode socks5
echo "* socks5"

./shapeshifter-dispatcher -mode socks5 -client -state state -transport shadow -proxylistenaddr 127.0.0.1:1443 -optionsFile shadowClient.json -logLevel DEBUG -enableLogging &

sleep 1

killall shapeshifter-dispatcher

# Test -mode STUN
echo "* STUN"

./shapeshifter-dispatcher -mode STUN -client -state state -transport shadow -proxylistenaddr 127.0.0.1:1443 -optionsFile shadowClient.json -logLevel DEBUG -enableLogging &

sleep 1

killall shapeshifter-dispatcher

echo "Testing complete. Killing processes."

killall shapeshifter-dispatcher
killall nc

echo "Done."
