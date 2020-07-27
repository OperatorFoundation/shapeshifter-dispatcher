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

package pt_extras

import (
	"encoding/json"
	"errors"
	options2 "github.com/OperatorFoundation/shapeshifter-dispatcher/common"
	"github.com/OperatorFoundation/shapeshifter-dispatcher/common/log"
	"github.com/OperatorFoundation/shapeshifter-dispatcher/transports"
	Optimizer "github.com/OperatorFoundation/shapeshifter-transports/transports/Optimizer/v3"
	"github.com/OperatorFoundation/shapeshifter-transports/transports/meekserver/v3"
	"github.com/kataras/golog"
	"golang.org/x/net/proxy"
	"net"


	"github.com/OperatorFoundation/shapeshifter-transports/transports/obfs2/v3"
	"github.com/OperatorFoundation/shapeshifter-transports/transports/obfs4/v3"
)

// target is the server address string
func ArgsToDialer(target string, name string, args string, dialer proxy.Dialer) (Optimizer.TransportDialer, error) {
	switch name {
	case "obfs2":
		transport := obfs2.New(target, dialer, log)
		return transport, nil
	case "obfs4":
		//refactor starts here
		transport, err := transports.ParseArgsObfs4(args, target, dialer)
		if err != nil {
			log.Errorf("Could not parse options %s", err.Error())
			return nil, err
		} else {
			return transport, nil
		}
	case "shadow":
		transport, err := transports.ParseArgsShadow(args, target)
		if err != nil {
			log.Errorf("Could not parse options %s", err.Error())
			return nil, err
		} else {
			return transport, nil
		}
	case "Optimizer":
		transport, err := transports.ParseArgsOptimizer(args, dialer)
		if err != nil {
			log.Errorf("Could not parse options %s", err.Error())
			return nil, err
		} else {
			return transport, nil
		}
	case "Dust":
		transport, err := transports.ParseArgsDust(args, target, dialer)
		if err != nil {
			log.Errorf("Could not parse options %s", err.Error())
			return nil, err
		} else {
			return transport, nil
		}
	case "meeklite":
		transport, err := transports.ParseArgsMeeklite(args, target, dialer)
		if err != nil {
			log.Errorf("Could not parse options %s", err.Error())
			return nil, err
		} else {
			return transport, nil
		}
	case "Replicant":
		transport, err := transports.ParseArgsReplicantClient(args, target, dialer)
		if err != nil {
			log.Errorf("Could not parse options %s", err.Error())
			return nil, err
		} else {
			return transport, nil
		}

	default:
		log.Errorf("Unknown transport: %s", name)
		return nil, errors.New("unknown transport")
	}
}

func ArgsToListener(name string, stateDir string, options string) (func(address string) (net.Listener, error), error) {
	var listen func(address string) (net.Listener, error)

	//var config meekserver.Config

	args, argsErr := options2.ParseServerOptions(options)
	if argsErr != nil {
		log.Errorf("Error parsing transport options: %s", options)
		return nil, errors.New("error parsing transport options")
	}

	switch name {
	case "obfs2":
		transport := obfs2.NewObfs2Transport()
		listen = transport.Listen
	case "obfs4":
		transport, err := obfs4.NewObfs4Server(stateDir)
		if err != nil {
			log.Errorf("Can't start obfs4 transport: %v", err)
			return nil, errors.New("can't start obfs4 transport")
		}
		listen = transport.Listen
	case "Replicant":
		shargs, aok := args["Replicant"]
		if !aok {
			return nil, errors.New("could not find Replicant options")
		}

		shargsBytes, err:= json.Marshal(shargs)
		if err != nil {
			return nil, errors.New("could not marshall json")
		}
		shargsString := string(shargsBytes)
		config, err := transports.ParseArgsReplicantServer(shargsString)
		if err != nil {
			return nil, errors.New("could not parse Replicant options")
		}

		//configJSONString, jsonMarshallError := json.Marshal(config)
		//if jsonMarshallError == nil {
		//	log.Debugf("REPLICANT CONFIG\n", string(configJSONString))
		//}

		return config.Listen, nil
	// FIXME - meeklite parsing is incorrect
	case "meekserver":
		shargs, aok := args["meekserver"]
		if !aok {
			return nil, errors.New("could not find meeklite options")
		}

		shargsByte, err:= json.Marshal(shargs)
		if err != nil {
			log.Errorf("could not coerce meeklite Url to string")
		}
		shargsString := string(shargsByte)
		config, err := transports.ParseArgsMeekliteServer(shargsString)
		if err != nil {
			return nil, errors.New("could not parse Replicant options")
		}
		transport := meekserver.NewMeekTransportServer(true, config.AcmeEmail, config.AcmeHostname, stateDir)
		listen = transport.Listen
	// FIXME - Dust parsing is incorrect
	//case "Dust":
	//	shargs, aok := args["Dust"]
	//	if !aok {
	//		return false
	//	}
	//
	//	untypedIdPath, ok := shargs["Url"]
	//	if !ok {
	//		return false
	//	}
	//	idPathByte, err:= json.Marshal(untypedIdPath)
	//	idPathString := string(idPathByte)
	//	if err != nil {
	//		log.Errorf("could not coerce Dust Url to string")
	//		return false
	//	}
	//	transport := Dust.NewDustServer(idPathString)
	//	listen = transport.Listen
	case "shadow":
		args, aok := args["shadow"]
		if !aok {
			return nil, errors.New("could not find shadow options")
		}

		argsBytes, err:= json.Marshal(args)
		argsString := string(argsBytes)
		config, err := transports.ParseArgsShadowServer(argsString)
		if err != nil {
			return nil, err
		}

		listen = config.Listen
	default:
		return nil, errors.New("unknown transport")
	}

	return listen, nil
}
