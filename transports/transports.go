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
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/OperatorFoundation/shapeshifter-dispatcher/common/log"
	"github.com/OperatorFoundation/shapeshifter-transports/transports/Dust"
	"github.com/OperatorFoundation/shapeshifter-transports/transports/Optimizer"
	replicant "github.com/OperatorFoundation/shapeshifter-transports/transports/Replicant"
	"github.com/OperatorFoundation/shapeshifter-transports/transports/Replicant/polish"
	"github.com/OperatorFoundation/shapeshifter-transports/transports/meeklite"
	"github.com/OperatorFoundation/shapeshifter-transports/transports/obfs4"
	"github.com/OperatorFoundation/shapeshifter-transports/transports/shadow"
	"github.com/mufti1/interconv/package"
	"golang.org/x/net/proxy"
	gourl "net/url"
	"strconv"
)

// Transports returns the list of registered transport protocols.
func Transports() []string {
	return []string{"obfs2", "shadow", "Dust", "meeklite", "Replicant", "obfs4", "Optimizer"}
}

func ParseArgsObfs4(args map[string]interface{}, target string, dialer proxy.Dialer) (*obfs4.Transport, error) {
	var cert string
	var iatMode int

	untypedCert, ok := args["cert"]
	if !ok {
		return nil, errors.New("obfs4 transport missing cert argument")
	}

	switch untypedCert.(type) {
	case string:
		var icerr error
		cert, icerr = interconv.ParseString(untypedCert)
		if icerr != nil {
			return nil, icerr
		}
	default:
		return nil, errors.New("unsupported type for obfs4 cert option")
	}

	untypedIatMode, ok2 := args["iat-mode"]
	if !ok2 {
		return nil, errors.New("obfs4 transport missing iat-mode argument")
	}

	switch untypedCert.(type) {
	case string:
		iatModeStr, icerr := interconv.ParseString(untypedIatMode)
		if icerr != nil {
			return nil, icerr
		}
		iatModeInt, scerr := strconv.Atoi(iatModeStr)
		if scerr != nil {
			return nil, errors.New("obfs4 transport bad iat-mode value")
		}
		switch iatModeInt {
		case 0:
			iatMode = iatModeInt
		case 1:
			iatMode = iatModeInt
		default:
			return nil, errors.New("unsupported value for obfs4 iat-mode option")
		}
	case float64:
		iatModeFloat, icerr := interconv.ParseFloat64(untypedIatMode)
		if icerr != nil {
			return nil, icerr
		}
		iatModeInt := int(iatModeFloat)
		switch iatModeInt {
		case 0:
			iatMode = iatModeInt
		case 1:
			iatMode = iatModeInt
		default:
			return nil, errors.New("unsupported value for obfs4 iat-mode option")
		}
	case int:
		iatModeInt, icerr := interconv.ParseInt(untypedIatMode)
		if icerr != nil {
			return nil, icerr
		}
		switch iatModeInt {
		case 0:
			iatMode = iatModeInt
		case 1:
			iatMode = iatModeInt
		default:
			return nil, errors.New("unsupported value for obfs4 iat-mode option")
		}
	case bool:
		iatModeBool, icerr := interconv.ParseBoolean(untypedCert)
		if icerr != nil {
			return nil, icerr
		}
		switch iatModeBool {
		case true:
			iatMode = 1
		case false:
			iatMode = 0
		}
	default:
		return nil, errors.New("unsupported type for obfs4 iat-mode option")
	}

	transport := obfs4.Transport{
		CertString: cert,
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

func ParseArgsDust(args map[string]interface{}, target string, dialer proxy.Dialer) (*Dust.Transport, error) {
	var serverPublic string

	untypedServerPublic, ok := args["serverPublic"]
	if !ok {
		return nil, errors.New("dust transport missing serverpublic argument")
	}

	switch untypedServerPublic.(type) {
	case string:
		var icerr error
		serverPublic, icerr = interconv.ParseString(untypedServerPublic)
		if icerr != nil {
			return nil, icerr
		}
	default:
		return nil, errors.New("unsupported type for dust serverpublic option")
	}

	transport := Dust.Transport{
		ServerPublic: serverPublic,
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
func ParseArgsReplicantServer(args map[string]interface{}) (*replicant.ServerConfig, error) {

	if args == nil {
		defaultConfig := CreateDefaultReplicantServer()
		configPointer := &defaultConfig
		return configPointer, nil
	}

	untypedConfig, ok := args["config"]
	if untypedConfig == nil {
		defaultConfig := CreateDefaultReplicantServer()
		configPointer := &defaultConfig
		return configPointer, nil
	}
	if !ok {
		return nil, errors.New("replicant transport missing config argument")
	}

	switch untypedConfig.(type) {
	case string:
		configString := untypedConfig.(string)

		var config *replicant.ServerConfig
		var parseErr error
		config, parseErr = replicant.DecodeServerConfig(configString)
		if parseErr != nil {
			return nil, errors.New("could not parse config")
		}

		return config, nil
	default:
		return nil, errors.New("unsupported type for replicant config option")
	}
}

func ParseArgsMeeklite(args map[string]interface{}, target string, dialer proxy.Dialer) (*meeklite.Transport, error) {

	var url *gourl.URL
	var front string

	untypedUrl, ok := args["url"]
	if !ok {
		return nil, errors.New("meeklite transport missing url argument")
	}

	switch untypedUrl.(type) {
	case string:

		urlString, icerr := interconv.ParseString(untypedUrl)
		if icerr != nil {
			return nil, icerr
		}
		var parseErr error
		url, parseErr = gourl.Parse(urlString)
		if parseErr != nil {
			return nil, errors.New("could not parse meeklite URL")
		}

	default:
		return nil, errors.New("unsupported type for meeklite url option")
	}

	untypedFront, ok2 := args["front"]
	if !ok2 {
		return nil, errors.New("meeklite transport missing front argument")
	}

	switch untypedFront.(type) {
	case string:
		var icerr error
		front, icerr = interconv.ParseString(untypedFront)
		if icerr != nil {
			return nil, icerr
		}
	default:
		return nil, errors.New("unsupported type for meeklite front option")
	}

	transport := meeklite.Transport{
		Url:     url,
		Front:   front,
		Address: target,
		Dialer:  dialer,
	}

	return &transport, nil
}

func ParseArgsOptimizer(args string, dialer proxy.Dialer) (*Optimizer.Client, error) {
	var transports []Optimizer.Transport
	var strategy Optimizer.Strategy

	untypedStrategy, ok2 := args["strategy"]
	if !ok2 {
		return nil, errors.New("optimizer transport missing strategy argument")
	}

	switch untypedStrategy.(type) {
	case string:
		strategyString, icerr := interconv.ParseString(untypedStrategy)
		if icerr != nil {
			return nil, icerr
		}
		var parseErr error
		strategy, parseErr = parseStrategy(strategyString, transports)
		if parseErr != nil {
			return nil, errors.New("could not parse strategy")
		}
	default:
		return nil, errors.New("unsupported type for optimizer strategy option")
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
		obfs4Transport, parseErr := ParseArgsObfs4(config, address, dialer)
		if parseErr != nil {
			return nil, errors.New("could not parse obfs4 Args")
		}
		return obfs4Transport, nil
	case "meeklite":
		meekliteTransport, parseErr := ParseArgsMeeklite(config, address, dialer)
		if parseErr != nil {
			return nil, errors.New("could not parse meeklite Args")
		}
		return meekliteTransport, nil
	case "Dust":
		DustTransport, parseErr := ParseArgsDust(config, address, dialer)
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
		optimizerTransport, parseErr := ParseArgsOptimizer(config, dialer)
		if parseErr != nil {
			return nil, errors.New("could not parse Optimizer Args")
		}
		return optimizerTransport, nil
	default:
		return nil, errors.New("unsupported transport name")
	}
}
