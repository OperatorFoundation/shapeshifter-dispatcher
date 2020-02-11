/*
 * Copyright (c) 2014, Yawning Angel <yawning at torproject dot org>
 * All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are met:
 *
 *  * Redistributions of source code must retain the above copyright notice,
 *    this list of conditions and the following disclaimer.
 *
 *  * Redistributions in binary form must reproduce the above copyright notice,
 *    this list of conditions and the following disclaimer in the documentation
 *    and/or other materials provided with the distribution.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
 * AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
 * IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
 * ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE
 * LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
 * CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
 * SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
 * INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
 * CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
 * ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
 * POSSIBILITY OF SUCH DAMAGE.
 */

// Package transports provides a interface to query supported pluggable
// transports.
package transports

import (
	"encoding/json"
	"errors"
	options "github.com/OperatorFoundation/shapeshifter-dispatcher/common"
	"github.com/OperatorFoundation/shapeshifter-transports/transports/Dust"
	"github.com/OperatorFoundation/shapeshifter-transports/transports/Optimizer"
	replicant "github.com/OperatorFoundation/shapeshifter-transports/transports/Replicant"
	"github.com/OperatorFoundation/shapeshifter-transports/transports/meeklite"
	"github.com/OperatorFoundation/shapeshifter-transports/transports/obfs4"
	"github.com/OperatorFoundation/shapeshifter-transports/transports/shadow"
	"golang.org/x/net/proxy"
)

// Transports returns the list of registered transport protocols.
func Transports() []string {
	return []string{"obfs2", "shadow", "Dust", "meeklite", "Replicant", "obfs4", "Optimizer"}
}

func ParseArgsObfs4(args string, target string, dialer proxy.Dialer) (*obfs4.Transport, error) {
	var config obfs4.Config

	bytes := []byte(args)
	jsonError := json.Unmarshal(bytes, &config)
	if jsonError != nil {
		return nil, errors.New("obfs4 options json decoding error")
	}

	iatMode := 0
	if config.IatMode == "1" {
		iatMode = 1
	}

	transport := obfs4.Transport{
		CertString: config.CertString,
		IatMode:    iatMode,
		Address:    target,
		Dialer:     dialer,
	}

	return &transport, nil
}

func ParseArgsShadow(args string, target string, dialer proxy.Dialer) (*shadow.Transport, error) {
	var config shadow.Config

	bytes := []byte(args)
	jsonError := json.Unmarshal(bytes, &config)
	if jsonError != nil {
		return nil, errors.New("shadow options json decoding error")
	}

	transport := shadow.Transport{
		Password:   config.Password,
		CipherName: config.CipherName,
		Address:    target,
		Dialer:     dialer,
	}

	return &transport, nil
}

func ParseArgsDust(args string, target string, dialer proxy.Dialer) (*Dust.Transport, error) {
	var config Dust.Config

	bytes := []byte(args)
	jsonError := json.Unmarshal(bytes, &config)
	if jsonError != nil {
		return nil, errors.New("dust options json decoding error")
	}

	transport := Dust.Transport{
		ServerPublic: config.ServerPublic,
		Address:      target,
		Dialer:       dialer,
	}

	return &transport, nil
}

func CreateDefaultReplicantClient(target string, dialer proxy.Dialer) (*replicant.Transport) {
	config := replicant.ClientConfig{
		Toneburst: nil,
		Polish:    nil,
	}

	transport := replicant.Transport{
		Config:  config,
		Address: target,
		Dialer:  dialer,
	}

	return &transport
}

func CreateDefaultReplicantServer() (replicant.ServerConfig) {
	config := replicant.ServerConfig{
		Toneburst: nil,
		Polish:    nil,
	}

	return config
}

func ParseArgsReplicantClient(args string, target string, dialer proxy.Dialer) (*replicant.Transport, error) {
	var config *replicant.ClientConfig

	type replicantJsonConfig struct {
		Config string
	}
	var ReplicantConfig replicantJsonConfig
	if args =="" {
		transport := CreateDefaultReplicantClient(target, dialer)
		return transport, nil
	}
	argsBytes := []byte(args)
	unmarshalError:= json.Unmarshal(argsBytes, &ReplicantConfig)
	if unmarshalError != nil {
		return nil, errors.New("could not unmarshal Replicant args")
	}
	var parseErr error
	config, parseErr = replicant.DecodeClientConfig(ReplicantConfig.Config)
	if parseErr != nil {
		return nil, errors.New("could not parse config")
	}

	transport := replicant.Transport{
		Config:  *config,
		Address: target,
		Dialer:  dialer,
	}

	return &transport, nil
}

//  target string, dialer proxy.Dialer
func ParseArgsReplicantServer(args string) (*replicant.ServerConfig, error) {
	var config *replicant.ServerConfig

	type replicantJsonConfig struct {
		Config string
	}
	var ReplicantConfig replicantJsonConfig
	if args =="" {
		transport := CreateDefaultReplicantServer()
		return &transport, nil
	}
	argsBytes := []byte(args)
	unmarshalError:= json.Unmarshal(argsBytes, &ReplicantConfig)
	if unmarshalError != nil {
		return nil, errors.New("could not unmarshal Replicant args")
	}
	var parseErr error
	config, parseErr = replicant.DecodeServerConfig(ReplicantConfig.Config)
	if parseErr != nil {
		return nil, errors.New("could not parse config")
	}
	return config, nil
}

func ParseArgsMeeklite(args string, target string, dialer proxy.Dialer) (*meeklite.Transport, error) {
	var config meeklite.Config

	bytes := []byte(args)
	jsonError := json.Unmarshal(bytes, &config)
	if jsonError != nil {
		return nil, errors.New("meeklite options json decoding error")
	}

	transport := meeklite.Transport{
		Url:     config.Url,
		Front:   config.Front,
		Address: target,
		Dialer:  dialer,
	}

	return &transport, nil
}

func ParseArgsOptimizer(jsonConfig string, dialer proxy.Dialer) (*Optimizer.Client, error) {
	var transports []Optimizer.Transport
	var strategy Optimizer.Strategy
	args, parseErr := options.ParseOptions(jsonConfig)
	if parseErr != nil {
		return nil, errors.New("could not marshal optimizer config")
	}
	//jsonBytes := []byte(jsonConfig)
	//unmarshalErr := json.Unmarshal(jsonBytes, &args)
	//if unmarshalErr != nil {
	//	return nil, errors.New("could not unmarshal optimizer config")
	//}

	untypedStrategy, ok2 := args["strategy"]
	if !ok2 {
		return nil, errors.New("optimizer transport missing strategy argument")
	}

	//FIXME if possible, replace CoerceToString with json parsing
	strategyString, icerr := options.CoerceToString(untypedStrategy)
	if icerr != nil {
		return nil, icerr
	}

	strategy, parseErr = parseStrategy(strategyString, transports)
	if parseErr != nil {
		return nil, errors.New("could not parse strategy")
	}

	untypedTransports, ok := args["transports"]
	if !ok {
		return nil, errors.New("optimizer transport missing transports argument")
	}

	switch untypedTransports.(type) {
	case []interface{}:
		otcs := untypedTransports.([]interface{})

		var parseErr error
		transports, parseErr = parseTransports(otcs, dialer)
		if parseErr != nil {
			return nil, errors.New("could not parse transports")
		}
	default:
		return nil, errors.New("unsupported type for Optimizer transports option")
	}

	transport := Optimizer.Client{
		Transports: transports,
		Strategy:   strategy,
	}

	return &transport, nil
}

func parseStrategy(strategyString string, transports []Optimizer.Transport) (Optimizer.Strategy, error) {
	switch strategyString {
	case "first":
		strategy := Optimizer.NewFirstStrategy(transports)
		return strategy, nil
	case "random":
		strategy := Optimizer.NewRandomStrategy(transports)
		return strategy, nil
	case "rotate":
		strategy := Optimizer.NewRotateStrategy(transports)
		return strategy, nil
	case "track":
		return Optimizer.NewTrackStrategy(transports), nil
	case "minimizeDialDuration":
		return Optimizer.NewMinimizeDialDuration(transports), nil

	default:
		return nil, errors.New("invalid strategy")
	}
}

func parseTransports(otcs []interface{}, dialer proxy.Dialer) ([]Optimizer.Transport, error) {
	transports := make([]Optimizer.Transport, len(otcs))
	for index, untypedOtc := range otcs {
		switch untypedOtc.(type) {
		case map[string]interface{}:
			otc := untypedOtc.(map[string]interface{})
			transport, err := parsedTransport(otc, dialer)
			if err != nil {
				return nil, errors.New("transport could not parse config")
				//this error sucks and is uninformative
			}
			transports[index] = transport
		default:
			return nil, errors.New("unsupported type for transport")
		}

	}
	return transports, nil
}

func parsedTransport(otc map[string]interface{}, dialer proxy.Dialer) (Optimizer.Transport, error) {
	var address string
	var name string
	var config map[string]interface{}

	type PartialOptimizerConfig struct {
		Address string
		Name    string
	}
	jsonString, MarshalErr := json.Marshal(otc)
	if MarshalErr != nil {
		return nil, errors.New("error marshalling optimizer otc")
	}
	var PartialConfig PartialOptimizerConfig
	unmarshalError := json.Unmarshal(jsonString, &PartialConfig)
	if unmarshalError != nil {
		return nil, errors.New("error unmarshalling optimizer otc")
	}
	//on to parsing the config
	untypedConfig, ok3 := otc["config"]
	if !ok3 {
		return nil, errors.New("missing config in transport parser")
	}

	switch untypedConfig.(type) {

	case map[string]interface{}:
		config = untypedConfig.(map[string]interface{})

	default:
		return nil, errors.New("unsupported type for optimizer config option")
	}

	jsonConfigBytes, configMarshalError:= json.Marshal(config)
	if configMarshalError != nil {
		return nil, errors.New("could not marshal Optimizer config")
	}
	jsonConfigString:= string(jsonConfigBytes)
	switch name {
	case "shadow":
		shadowTransport, parseErr := ParseArgsShadow(jsonConfigString, address, dialer)
		if parseErr != nil {
			return nil, errors.New("could not parse shadow Args")
		}
		return shadowTransport, nil
	case "obfs4":
		obfs4Transport, parseErr := ParseArgsObfs4(jsonConfigString, address, dialer)
		if parseErr != nil {
			return nil, errors.New("could not parse obfs4 Args")
		}
		return obfs4Transport, nil
	case "meeklite":
		meekliteTransport, parseErr := ParseArgsMeeklite(jsonConfigString, address, dialer)
		if parseErr != nil {
			return nil, errors.New("could not parse meeklite Args")
		}
		return meekliteTransport, nil
	case "Dust":
		DustTransport, parseErr := ParseArgsDust(jsonConfigString, address, dialer)
		if parseErr != nil {
			return nil, errors.New("could not parse dust Args")
		}
		return DustTransport, nil
	case "Replicant":
		replicantTransport, parseErr := ParseArgsReplicantClient(jsonConfigString, address, dialer)
		if parseErr != nil {
			return nil, errors.New("could not parse replicant Args")
		}
		return replicantTransport, nil
	case "Optimizer":
		optimizerTransport, parseErr := ParseArgsOptimizer(jsonConfigString, dialer)
		if parseErr != nil {
			return nil, errors.New("could not parse Optimizer Args")
		}
		return optimizerTransport, nil
	default:
		return nil, errors.New("unsupported transport name")
	}
}
