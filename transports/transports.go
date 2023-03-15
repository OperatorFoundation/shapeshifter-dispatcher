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
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"strings"

	Optimizer "github.com/OperatorFoundation/Optimizer-go/Optimizer/v3"
	replicant "github.com/OperatorFoundation/Replicant-go/Replicant/v3"
	"github.com/OperatorFoundation/Replicant-go/Replicant/v3/toneburst"
	"github.com/OperatorFoundation/Shadow-go/shadow/v3"
	"github.com/OperatorFoundation/Starbridge-go/Starbridge/v3"
	shadowsocks "github.com/OperatorFoundation/go-shadowsocks2/darkstar"
	"github.com/aead/ecdh"
	"github.com/kataras/golog"
	"golang.org/x/net/proxy"
)

// Transports returns the list of registered transport protocols.
func Transports() []string {
	return []string{"shadow", "Replicant", "Starbridge", "Optimizer"}
}

func ParseArgsShadow(args string, enableLocket bool, logDir string) (*shadow.Transport, error) {
	var config shadow.ClientConfig

	if enableLocket {
		config.LogDir = &logDir
	} else {
		config.LogDir = nil
	}

	bytes := []byte(args)
	jsonError := json.Unmarshal(bytes, &config)
	if jsonError != nil {
		return nil, errors.New("shadow options json decoding error")
	}
	transport := shadow.NewTransport(config.ServerAddress, config.ServerPublicKey, config.CipherName, config.LogDir)

	return &transport, nil
}

func ParseArgsShadowServer(args string, enableLocket bool, logDir string) (*shadow.ServerConfig, error) {
	var config shadow.ServerConfig

	if enableLocket {
		config.LogDir = &logDir
	} else {
		config.LogDir = nil
	}

	bytes := []byte(args)
	jsonError := json.Unmarshal(bytes, &config)
	if jsonError != nil {
		return nil, errors.New("shadow server options json decoding error")
	}

	return &config, nil
}

func CreateDefaultReplicantServer() replicant.ServerConfig {
	config := replicant.ServerConfig{
		Toneburst: nil,
		Polish:    nil,
	}

	return config
}

func ParseArgsReplicantClient(args string, dialer proxy.Dialer) (*replicant.TransportClient, error) {
	config, jsonError := replicant.UnmarshalClientConfig([]byte(args))
	if jsonError != nil {
		return nil, jsonError
	}

	transport := replicant.TransportClient{
		Config:  *config,
		Address: config.ServerAddress,
		Dialer:  dialer,
	}

	return &transport, nil
}

//  target string, dialer proxy.Dialer
func ParseArgsReplicantServer(args string) (*replicant.ServerConfig, error) {
	config, jsonError := replicant.UnmarshalServerConfig([]byte(args))
	if jsonError != nil {
		return nil, jsonError
	}

	return config, nil
}

func ParseArgsStarbridgeClient(args string, dialer proxy.Dialer) (*Starbridge.TransportClient, error) {
	var config Starbridge.ClientConfig
	bytes := []byte(args)
	jsonError := json.Unmarshal(bytes, &config)
	if jsonError != nil {
		return nil, errors.New("starbridge client options json decoding error")
	}

	transport := Starbridge.TransportClient{
		Config:  config,
		Address: config.ServerAddress,
		Dialer:  dialer,
	}

	return &transport, nil
}

func ParseArgsStarbridgeServer(args string) (*Starbridge.ServerConfig, error) {
	var config Starbridge.ServerConfig

	bytes := []byte(args)
	jsonError := json.Unmarshal(bytes, &config)
	if jsonError != nil {
		return nil, errors.New("starbridge server options json decoding error")
	}

	return &config, nil
}

type OptimizerConfig struct {
	Transports []interface{} `json:"transports"`
	Strategy   string        `json:"strategy"`
}

type OptimizerArgs struct {
	Address string                 `json:"address"`
	Name    string                 `json:"name"`
	Config  map[string]interface{} `json:"config"`
}

func ParseArgsOptimizer(jsonConfig string, dialer proxy.Dialer, enableLocket bool, logDir string) (*Optimizer.Client, error) {
	var config OptimizerConfig
	var transports []Optimizer.TransportDialer
	var strategy Optimizer.Strategy
	jsonByte := []byte(jsonConfig)
	parseErr := json.Unmarshal(jsonByte, &config)
	if parseErr != nil {
		return nil, errors.New("could not marshal optimizer config")
	}
	transports, parseErr = parseTransports(config.Transports, dialer, enableLocket, logDir)
	if parseErr != nil {
		println("this is the returned error from parseTransports:", parseErr)
		return nil, errors.New("could not parse transports")
	}

	strategy, parseErr = parseStrategy(config.Strategy, transports)
	if parseErr != nil {
		return nil, errors.New("could not parse strategy")
	}

	transport := Optimizer.NewOptimizerClient(transports, strategy)

	return transport, nil
}

func parseStrategy(strategyString string, transports []Optimizer.TransportDialer) (Optimizer.Strategy, error) {
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

func parseTransports(otcs []interface{}, dialer proxy.Dialer, enableLocket bool, logDir string) ([]Optimizer.TransportDialer, error) {
	transports := make([]Optimizer.TransportDialer, len(otcs))
	for index, untypedOtc := range otcs {
		switch untypedOtc.(type) {
		case map[string]interface{}:
			otc := untypedOtc.(map[string]interface{})
			transport, err := parsedTransport(otc, dialer, enableLocket, logDir)
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

func parsedTransport(otc map[string]interface{}, dialer proxy.Dialer, enableLocket bool, logDir string) (Optimizer.TransportDialer, error) {
	var config map[string]interface{}

	type PartialOptimizerConfig struct {
		Name string `json:"name"`
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

	jsonConfigBytes, configMarshalError := json.Marshal(config)
	if configMarshalError != nil {
		return nil, errors.New("could not marshal Optimizer config")
	}
	jsonConfigString := string(jsonConfigBytes)
	switch strings.ToLower(PartialConfig.Name) {
	case "shadow":
		shadowTransport, parseErr := ParseArgsShadow(jsonConfigString, enableLocket, logDir)
		if parseErr != nil {
			return nil, errors.New("could not parse shadow Args")
		}
		return shadowTransport, nil
	case "replicant":
		replicantTransport, parseErr := ParseArgsReplicantClient(jsonConfigString, dialer)
		if parseErr != nil {
			return nil, errors.New("could not parse replicant Args")
		}
		return replicantTransport, nil
	case "starbridge":
		starbridgeTransport, parseErr := ParseArgsStarbridgeClient(jsonConfigString, dialer)
		if parseErr != nil {
			return nil, errors.New("could not parse starbridge Args")
		}
		return starbridgeTransport, nil
	case "optimizer":
		optimizerTransport, parseErr := ParseArgsOptimizer(jsonConfigString, dialer, enableLocket, logDir)
		if parseErr != nil {
			return nil, errors.New("could not parse Optimizer Args")
		}
		return optimizerTransport, nil
	default:
		println("unsupported transport name")
		return nil, errors.New("unsupported transport name")
	}
}

func CreateShadowConfigs(address string) error {
	keyExchange := ecdh.Generic(elliptic.P256())
	clientEphemeralPrivateKey, clientEphemeralPublicKeyPoint, keyError := keyExchange.GenerateKey(rand.Reader)
	if keyError != nil {
		return keyError
	}

	privateKeyBytes, ok := clientEphemeralPrivateKey.([]byte)
	if !ok {
		return errors.New("could not convert private key to bytes")
	}

	publicKeyBytes, keyByteError := shadowsocks.PublicKeyToBytes(clientEphemeralPublicKeyPoint)
	if keyByteError != nil {
		return keyByteError
	}

	privateKeyString := base64.StdEncoding.EncodeToString(privateKeyBytes)
	publicKeyString := base64.StdEncoding.EncodeToString(publicKeyBytes)

	shadowServerConfig := shadow.ServerConfig{
		ServerAddress: 	  address,
		ServerPrivateKey: privateKeyString,
		CipherName:		  "darkstar",
		Transport: 		  "Shadow",
	}

	serverJsonBytes, marshalError := json.MarshalIndent(shadowServerConfig, "", "  ")
	if marshalError != nil {
		return marshalError
	}

	shadowClientConfig := shadow.ClientConfig{
		ServerAddress:   address,
		ServerPublicKey: publicKeyString,
		CipherName: 	 "darkstar",
		Transport: 		 "Shadow",
	}

	clientJsonBytes, marshalError := json.MarshalIndent(shadowClientConfig, "", "  ")
	if marshalError != nil {
		return marshalError
	}

	serverJsonError := os.WriteFile("ShadowServerConfig.json", serverJsonBytes, 0777)
	if serverJsonError != nil {
		return serverJsonError
	}
	
	clientJsonError := os.WriteFile("ShadowClientConfig.json", clientJsonBytes, 0777)
	if clientJsonError != nil {
		return clientJsonError
	}

	return nil
}

func CreateStarbridgeConfigs(address string) error {
	keyExchange := ecdh.Generic(elliptic.P256())
	clientEphemeralPrivateKey, clientEphemeralPublicKeyPoint, keyError := keyExchange.GenerateKey(rand.Reader)
	if keyError != nil {
		return keyError
	}

	privateKeyBytes, ok := clientEphemeralPrivateKey.([]byte)
	if !ok {
		return errors.New("could not convert private key to bytes")
	}

	publicKeyBytes, keyByteError := shadowsocks.PublicKeyToBytes(clientEphemeralPublicKeyPoint)
	if keyByteError != nil {
		return keyByteError
	}

	privateKeyString := base64.StdEncoding.EncodeToString(privateKeyBytes)
	publicKeyString := base64.StdEncoding.EncodeToString(publicKeyBytes)

	starbridgeClientConfig := Starbridge.ClientConfig {
		ServerAddress: address,
		ServerPublicKey: publicKeyString,
		Transport: "Starbridge",
	}

	starbridgeServerConfig := Starbridge.ServerConfig {
		ServerAddress: address,
		ServerPrivateKey: privateKeyString,
		Transport: "Starbridge",
	}

	serverJsonBytes, marshalError := json.MarshalIndent(starbridgeServerConfig, "", "  ")
	if marshalError != nil {
		return marshalError
	}

	clientJsonBytes, marshalError := json.MarshalIndent(starbridgeClientConfig, "", "  ")
	if marshalError != nil {
		return marshalError
	}

	serverJsonError := os.WriteFile("StarbridgeServerConfig.json", serverJsonBytes, 0777)
	if serverJsonError != nil {
		return serverJsonError
	}
	
	clientJsonError := os.WriteFile("StarbridgeClientConfig.json", clientJsonBytes, 0777)
	if clientJsonError != nil {
		return clientJsonError
	}

	return nil
}

func CreateReplicantConfigs(address string, isToneburst bool, isPolish bool) error {
	var polishClient *replicant.DarkStarPolishClientJsonConfig = nil 
	var polishServer *replicant.DarkStarPolishServerJsonConfig = nil 
	var toneburstClient *toneburst.StarburstConfig = nil
	var toneburstServer *toneburst.StarburstConfig = nil
	if isPolish {
		keyExchange := ecdh.Generic(elliptic.P256())
		clientEphemeralPrivateKey, clientEphemeralPublicKeyPoint, keyError := keyExchange.GenerateKey(rand.Reader)
		if keyError != nil {
			return keyError
		}

		privateKeyBytes, ok := clientEphemeralPrivateKey.([]byte)
		if !ok {
			return errors.New("could not convert private key to bytes")
		}

		publicKeyBytes, keyByteError := shadowsocks.PublicKeyToBytes(clientEphemeralPublicKeyPoint)
		if keyByteError != nil {
			return keyByteError
		}

		privateKeyString := base64.StdEncoding.EncodeToString(privateKeyBytes)
		publicKeyString := base64.StdEncoding.EncodeToString(publicKeyBytes)

		polishClient = &replicant.DarkStarPolishClientJsonConfig {
			ServerPublicKey: publicKeyString,
		}

		polishServer = &replicant.DarkStarPolishServerJsonConfig {
			ServerPrivateKey: privateKeyString,
		}

	} else {
		golog.Info("Invalid polish name.  Setting value to nil")
		polishClient = nil
		polishServer = nil
	}


	if isToneburst {
		toneburstClient = &toneburst.StarburstConfig{
			Mode: "SMTPClient",
		}

		toneburstServer = &toneburst.StarburstConfig{
			Mode: "SMTPServer",
		}
	}  else {
		golog.Info("Invalid toneburst name.  Setting value to nil")
			toneburstClient = nil
			toneburstServer = nil
	}
	
	replicantServerConfig := replicant.ServerJsonConfig {
		ServerAddress: address,
		Toneburst: *toneburstServer,
		Polish: *polishServer,
		Transport: "Replicant",
	}
	
	replicantClientConfig := replicant.ClientJsonConfig {
		ServerAddress: address,
		Toneburst: *toneburstClient,
		Polish: *polishClient,
		Transport: "Replicant",
	}

	serverJsonBytes, marshalError := json.MarshalIndent(replicantServerConfig, "", "  ")
	if marshalError != nil {
		return marshalError
	}

	clientJsonBytes, marshalError := json.MarshalIndent(replicantClientConfig, "", "  ")
	if marshalError != nil {
		return marshalError
	}

	serverJsonError := os.WriteFile("ReplicantServerConfig.json", serverJsonBytes, 0777)
	if serverJsonError != nil {
		return serverJsonError
	}
	
	clientJsonError := os.WriteFile("ReplicantClientConfig.json", clientJsonBytes, 0777)
	if clientJsonError != nil {
		return clientJsonError
	}

	return nil
}
