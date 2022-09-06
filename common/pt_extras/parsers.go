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
	Optimizer "github.com/OperatorFoundation/Optimizer-go/Optimizer/v3"
	options2 "github.com/OperatorFoundation/shapeshifter-dispatcher/common"
	"github.com/OperatorFoundation/shapeshifter-dispatcher/transports"
	"github.com/kataras/golog"
	"golang.org/x/net/proxy"
	"net"
)

// target is the server address string
func ArgsToDialer(name string, args string, dialer proxy.Dialer, enableLocket bool, logDir string) (Optimizer.TransportDialer, error) {
	switch name {
	case "shadow":
		transport, err := transports.ParseArgsShadow(args, enableLocket, logDir)
		if err != nil {
			golog.Errorf("Could not parse options %s", err.Error())
			return nil, err
		} else {
			return transport, nil
		}
	case "Optimizer":
		transport, err := transports.ParseArgsOptimizer(args, dialer, enableLocket, logDir)
		if err != nil {
			golog.Errorf("Could not parse options %s", err.Error())
			return nil, err
		} else {
			return transport, nil
		}
	case "Replicant":
		transport, err := transports.ParseArgsReplicantClient(args, dialer)
		if err != nil {
			golog.Errorf("Could not parse options %s", err.Error())
			return nil, err
		} else {
			return transport, nil
		}
	case "Starbridge":
		transport, err := transports.ParseArgsStarbridgeClient(args, dialer)
		if err != nil {
			golog.Errorf("Could not parse options %s", err.Error())
			return nil, err
		} else {
			return transport, nil
		}

	default:
		golog.Errorf("Unknown transport: %s", name)
		return nil, errors.New("unknown transport")
	}
}

func ArgsToListener(name string, stateDir string, options string, enableLocket bool, logDir string) (func(address string) (net.Listener, error), error) {
	var listen func(address string) (net.Listener, error)

	//var config meekserver.Config

	args, argsErr := options2.ParseServerOptions(options)
	if argsErr != nil {
		golog.Errorf("Error parsing transport options: %s", options)
		return nil, errors.New("error parsing transport options")
	}

	switch name {
	case "Replicant":
		shargs, aok := args["Replicant"]
		if !aok {
			return nil, errors.New("could not find Replicant options")
		}

		shargsBytes, err := json.Marshal(shargs)
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
	case "Starbridge":
		shargs, aok := args["Starbridge"]
		if !aok {
			return nil, errors.New("could not find Starbridge options")
		}

		shargsBytes, err := json.Marshal(shargs)
		shargsString := string(shargsBytes)
		config, err := transports.ParseArgsStarbridgeServer(shargsString)
		if err != nil {
			return nil, errors.New("could not parse Starbridge options")
		}

		return config.Listen, nil
	case "shadow":
		args, aok := args["shadow"]
		if !aok {
			return nil, errors.New("could not find shadow options")
		}

		argsBytes, err := json.Marshal(args)
		argsString := string(argsBytes)
		config, err := transports.ParseArgsShadowServer(argsString, enableLocket, logDir)
		if err != nil {
			return nil, err
		}

		listen = config.Listen
	default:
		return nil, errors.New("unknown transport")
	}

	return listen, nil
}
