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
	"errors"
	"fmt"
	"github.com/OperatorFoundation/shapeshifter-dispatcher/common/pt_extras"
	pt "github.com/OperatorFoundation/shapeshifter-ipc/v2"
	"github.com/op/go-logging"
	"io"
	"net"
	"net/url"
	"os"
	"sync"

	"github.com/OperatorFoundation/shapeshifter-dispatcher/common/log"
)

func ClientSetupTCP(socksAddr string, target string, ptClientProxy *url.URL, names []string, options string, clientHandler ClientHandlerTCP, log *logging.Logger) (launched bool) {
	// Launch each of the client listeners.
	for _, name := range names {
		ln, err := net.Listen("tcp", socksAddr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to listen %s %s", name, err.Error());
			log.Errorf("failed to listen %s %s", name, err.Error())
			continue
		}

		go clientAcceptLoop(target, name, options, ln, ptClientProxy, clientHandler, log)
		log.Infof("%s - registered listener: %s", name, ln.Addr())
		launched = true
	}

	return
}

func clientAcceptLoop(target string, name string, options string, ln net.Listener, proxyURI *url.URL, clientHandler ClientHandlerTCP, log *logging.Logger) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			if e, ok := err.(net.Error); ok && !e.Temporary() {
				fmt.Fprintf(os.Stderr, "Fatal listener error: %s", err.Error());
				log.Errorf("Fatal listener error: %s", err.Error())
				return
			}
			log.Warningf("Failed to accept connection: %s", err.Error())
			continue
		}

		go clientHandler(target, name, options, conn, proxyURI, log)
	}
}

func ServerSetupTCP(ptServerInfo pt.ServerInfo, stateDir string, options string, serverHandler ServerHandler) (launched bool) {
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
				print("listening on ")
				println(bindaddr.Addr.String())
				transportLn, LnError := listen(bindaddr.Addr.String())
				if LnError != nil {
					continue
				}
				log.Infof("%s - registered listener: %s", name, log.ElideAddr(bindaddr.Addr.String()))
				ServerAcceptLoop(name, transportLn, &ptServerInfo, serverHandler)
				transportLnErr := transportLn.Close()
				if transportLnErr != nil {
					fmt.Fprintf(os.Stderr, "Listener close error: %s", transportLnErr.Error());
					log.Errorf("Listener close error: %s", transportLnErr.Error())
				}
			}
		}()

		launched = true
	}

	return
}

func CopyLoop(a net.Conn, b net.Conn) error {
	println("--> Entering copy loop.")
	// Note: b is always the pt connection.  a is the SOCKS/ORPort connection.
	errChan := make(chan error, 2)

	var wg sync.WaitGroup
	wg.Add(2)

	if b == nil {
		println("--> Copy loop has a nil connection (b).")
		return errors.New("copy loop has a nil connection (b)")
	}

	if a == nil {
		println("--> Copy loop has a nil connection (a).")
		return errors.New("copy loop has a nil connection (a)")
	}

	go func() {
		defer wg.Done()
		_, err := io.Copy(b, a)
		errChan <- err
	}()
	go func() {
		defer wg.Done()
		_, err := io.Copy(a, b)
		errChan <- err
	}()

	// Wait for both upstream and downstream to close.  Since one side
	// terminating closes the other, the second error in the channel will be
	// something like EINVAL (though io.Copy() will swallow EOF), so only the
	// first error is returned.
	wg.Wait()
	if len(errChan) > 0 {
		return <-errChan
	}

	return nil
}
