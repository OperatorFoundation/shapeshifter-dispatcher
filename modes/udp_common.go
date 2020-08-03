/*
MIT License

Copyright (c) 2020 Operator Foundation

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NON-INFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package modes

import (
	"github.com/OperatorFoundation/obfs4/common/log"
	"github.com/OperatorFoundation/shapeshifter-dispatcher/common/pt_extras"
	pt "github.com/OperatorFoundation/shapeshifter-ipc/v2"
	"net"
	"net/url"
)

func ClientSetupUDP(socksAddr string, ptClientProxy *url.URL, names []string, options string, clientHandler ClientHandlerUDP) bool {
	// Launch each of the client listeners.
	for _, name := range names {
		udpAddr, err := net.ResolveUDPAddr("udp", socksAddr)
		if err != nil {
			log.Errorf("Error resolving address %s", socksAddr)
		}

		ln, err := net.ListenUDP("udp", udpAddr)
		if err != nil {
			log.Errorf("failed to listen %s %s", name, err.Error())
			continue
		}

		log.Infof("%s - registered listener", name)

		go clientHandler(name, options, ln, ptClientProxy)
	}

	return true
}

func ServerSetupUDP(ptServerInfo pt.ServerInfo, stateDir string, options string, serverHandler ServerHandler) (launched bool) {
	// Launch each of the server listeners.
	for _, bindaddr := range ptServerInfo.Bindaddrs {
		name := bindaddr.MethodName

		// Deal with arguments.
		listen, parseError := pt_extras.ArgsToListener(name, stateDir, options)
		if parseError != nil {
			return false
		}


		go func() {
			for {
				transportLn, LnError := listen(bindaddr.Addr.String())
				if LnError != nil {
					continue
				}
				log.Infof("%s - registered listener: %s", name, log.ElideAddr(bindaddr.Addr.String()))
				ServerAcceptLoop(name, transportLn, &ptServerInfo, serverHandler)
				transportLnErr := transportLn.Close()
				if transportLnErr != nil {
					log.Errorf("Listener close error: %s", transportLnErr.Error())
				}
			}
		}()

		launched = true
	}

	return
}
