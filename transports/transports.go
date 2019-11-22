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
	"errors"
	"github.com/OperatorFoundation/shapeshifter-dispatcher/common/log"
	"github.com/OperatorFoundation/shapeshifter-transports/transports/Dust"
	"github.com/OperatorFoundation/shapeshifter-transports/transports/Optimizer"
	replicant "github.com/OperatorFoundation/shapeshifter-transports/transports/Replicant"
	"github.com/OperatorFoundation/shapeshifter-transports/transports/Replicant/polish"
	"github.com/OperatorFoundation/shapeshifter-transports/transports/Replicant/toneburst"
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

func ParseArgsShadow(args map[string]interface{}, target string, dialer proxy.Dialer) (*shadow.Transport, error) {
	var password string
	var cipherName string

	untypedPassword, ok := args["password"]
	if !ok {
		return nil, errors.New("shadow transport missing password argument")
	}

	switch untypedPassword.(type) {
	case string:
		var icerr error
		password, icerr = interconv.ParseString(untypedPassword)
		if icerr != nil {
			return nil, icerr
		}
	default:
		return nil, errors.New("unsupported type for shadow password option")
	}

	untypedCipherName, ok2 := args["cipherName"]
	if !ok2 {
		return nil, errors.New("shadow transport missing cipherName argument")
	}

	switch untypedCipherName.(type) {
	case string:
		var icerr error
		cipherName, icerr = interconv.ParseString(untypedCipherName)
		if icerr != nil {
			return nil, icerr
		}
	default:
		return nil, errors.New("unsupported type for shadow cipherName option")
	}

	transport := shadow.Transport{
		Password:   password,
		CipherName: cipherName,
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

func ParseArgsReplicant(args map[string]interface{}, target string, dialer proxy.Dialer) (*replicant.Transport, error) {
	var config *replicant.Config
	untypedConfig, ok := args["config"]
	if !ok {
		return nil, errors.New("replicant transport missing config argument")
	}

	switch untypedConfig.(type) {
	case map[string]interface{}:
		otcs := untypedConfig.(map[string]interface{})

		var parseErr error
		config, parseErr = ParseReplicantConfig(otcs)
		if parseErr != nil {
			return nil, errors.New("could not parse config")
		}
	default:
		return nil, errors.New("unsupported type for replicant config option")
	}

	transport := replicant.Transport{
		Config:  *config,
		Address: target,
		Dialer:  dialer,
	}

	return &transport, nil
}

func ParseReplicantConfig(args map[string]interface{}) (*replicant.Config, error) {
	var toneburstConfig *toneburst.Config
	var polishConfig *polish.Config
	untypedToneburstConfig, ok := args["toneburst"]
	if !ok {
		return nil, errors.New("replicant transport missing toneburst argument")
	}

	switch untypedToneburstConfig.(type) {
	case map[string]interface{}:
		otcs := untypedToneburstConfig.(map[string]interface{})

		var parseErr error
		toneburstConfig, parseErr = parseToneburstConfig(otcs)
		if parseErr != nil {
			return nil, errors.New("could not parse replicant toneburst config")
		}
	default:
		return nil, errors.New("unsupported type for replicant toneburst option")
	}
	untypedPolishConfig, ok := args["polish"]
	if !ok {
		return nil, errors.New("replicant transport missing polish argument")
	}

	switch untypedPolishConfig.(type) {
	case map[string]interface{}:
		otcs := untypedPolishConfig.(map[string]interface{})
		var parseErr error
		polishConfig, parseErr = parsePolishConfig(otcs)
		if parseErr != nil {
			return nil, errors.New("could not parse replicant polish config")
		}
	default:
		return nil, errors.New("unsupported type for replicant polish option")
	}

	replicantConfig := replicant.Config{
		Toneburst: toneburstConfig,
		Polish:    polishConfig,
	}

	return &replicantConfig, nil
}

func parseToneburstConfig(args map[string]interface{}) (*toneburst.Config, error) {
	var selector string

	untypedSelector, ok := args["selector"]
	if !ok {
		return nil, errors.New("replicant toneburst missing selector argument")
	}

	switch untypedSelector.(type) {
	case string:
		var icerr error
		selector, icerr = interconv.ParseString(untypedSelector)
		if icerr != nil {
			return nil, icerr
		}
	default:
		return nil, errors.New("unsupported type for replicant toneburst selector option")
	}
	switch selector {
	case "whalesong":
		untypedWhalesongConfig, ok := args["whalesong"]
		if !ok {
			return nil, errors.New("replicant transport missing replicant toneburst whalesong argument")
		}

		switch untypedWhalesongConfig.(type) {
		case map[string]interface{}:
			otcs := untypedWhalesongConfig.(map[string]interface{})

			var parseErr error
			whalesongConfig, parseErr := parseWhalesongConfig(otcs)
			if parseErr != nil {
				return nil, errors.New("could not parse replicant toneburst whalesong config")
			}

			toneburstConfig := toneburst.Config{
				Selector:  selector,
				Whalesong: whalesongConfig,
			}

			return &toneburstConfig, nil
		default:
			return nil, errors.New("unsupported type for replicant toneburst whalesong option")
		}
	default:
		return nil, errors.New("unsupported value for replicant toneburst selector")
	}

}

func parsePolishConfig(args map[string]interface{}) (*polish.Config, error) {
	var selector string

	untypedSelector, ok := args["selector"]
	if !ok {
		return nil, errors.New("replicant polish missing selector argument")
	}

	switch untypedSelector.(type) {
	case string:
		var icerr error
		selector, icerr = interconv.ParseString(untypedSelector)
		if icerr != nil {
			return nil, icerr
		}
	default:
		return nil, errors.New("unsupported type for replicant polish selector option")
	}
	switch selector {
	case "silver":
		untypedSilverConfig, ok := args["silver"]
		if !ok {
			return nil, errors.New("replicant transport missing replicant polish silver argument")
		}

		switch untypedSilverConfig.(type) {
		case map[string]interface{}:
			otcs := untypedSilverConfig.(map[string]interface{})

			var parseErr error
			silverConfig, parseErr := parseSilverConfig(otcs)
			if parseErr != nil {
				return nil, errors.New("could not parse replicant polish silver config")
			}

			polishConfig := polish.Config{
				Selector: selector,
				Silver:   silverConfig,
			}

			return &polishConfig, nil
		default:
			return nil, errors.New("unsupported type for replicant polish silver option")
		}
	default:
		return nil, errors.New("unsupported value for replicant polish selector")
	}

}

func parseWhalesongConfig(args map[string]interface{}) (*toneburst.WhalesongConfig, error) {
	var addSequences []toneburst.Sequence
	var removeSequences []toneburst.Sequence

	untypedAddSequences, ok := args["add"]
	if !ok {
		return nil, errors.New("replicant transport missing toneburst whalesong add argument")
	}

	switch untypedAddSequences.(type) {
	case []interface{}:
		otcs := untypedAddSequences.([]interface{})
		addSequences = make([]toneburst.Sequence, len(otcs))
		for index, otc := range otcs {
			var sequence toneburst.Sequence
			sequenceString, icError := interconv.ParseString(otc)
			if icError != nil {
				log.Errorf("could not parse add sequence string")
			}
			bytes, byteErr := hex.DecodeString(sequenceString)
			if byteErr != nil {
				log.Errorf("could not parse add sequence string bytes")
			}
			sequence = bytes
			addSequences[index] = sequence
		}
	default:
		return nil, errors.New("unsupported type for replicant toneburst whalesong add option")
	}

	untypedRemoveSequences, ok := args["remove"]
	if !ok {
		return nil, errors.New("replicant transport missing toneburst whalesong remove argument")
	}

	switch untypedRemoveSequences.(type) {
	case []interface{}:
		otcs := untypedRemoveSequences.([]interface{})
		removeSequences = make([]toneburst.Sequence, len(otcs))
		for index, otc := range otcs {
			var sequence toneburst.Sequence
			sequenceString, icError := interconv.ParseString(otc)
			if icError != nil {
				log.Errorf("could not parse remove sequence string")
			}
			bytes, byteErr := hex.DecodeString(sequenceString)
			if byteErr != nil {
				log.Errorf("could not parse remove sequence string bytes")
			}
			sequence = bytes
			removeSequences[index] = sequence
		}
	default:
		return nil, errors.New("unsupported type for replicant toneburst whalesong option")
	}

	whalesongConfig := toneburst.WhalesongConfig{
		AddSequences:    addSequences,
		RemoveSequences: removeSequences,
	}

	return &whalesongConfig, nil
}

func parseSilverConfig(args map[string]interface{}) (*polish.SilverPolishConfig, error) {
	var clientOrServer bool
	var clientConfig *polish.SilverPolishClientConfig
	var serverConfig *polish.SilverPolishServerConfig

	untypedClientOrServer, ok := args["client or server"]
	if !ok {
		return nil, errors.New("replicant transport missing Silver Polish Client or Server argument")
	}

	switch untypedClientOrServer.(type) {
	case bool:
		var icerr error
		clientOrServer, icerr = interconv.ParseBoolean(untypedClientOrServer)
		if icerr != nil {
			return nil, icerr
		}
	default:
		return nil, errors.New("unsupported type for replicant Silver Polish Client or Server option")
	}

	switch clientOrServer {
	case true:
		untypedClientConfig, ok := args["clientConfig"]
		if !ok {
			return nil, errors.New("replicant transport missing Silver Polish clientConfig argument")
		}

		switch untypedClientConfig.(type) {
		case map[string]interface{}:
			otcs := untypedClientConfig.(map[string]interface{})

			var parseErr error
			clientConfig, parseErr = parseClientConfig(otcs)
			if parseErr != nil {
				return nil, errors.New("could not parse clientConfig")
			}
		}
	case false:
		untypedServerConfig, ok2 := args["serverConfig"]
		if !ok2 {
			return nil, errors.New("replicant transport Silver Polish missing ServerConfig")
		}
		switch untypedServerConfig.(type) {
		case map[string]interface{}:
			otcs := untypedServerConfig.(map[string]interface{})

			var parseErr error
			serverConfig, parseErr = parseServerConfig(otcs)
			if parseErr != nil {
				return nil, errors.New("could not parse Silver Polish serverConfig")
			}
		}
	}

	silverConfig := polish.SilverPolishConfig{
		ClientOrServer: clientOrServer,
		ClientConfig:   clientConfig,
		ServerConfig:   serverConfig,
	}

	return &silverConfig, nil
}
func parseClientConfig(args map[string]interface{}) (*polish.SilverPolishClientConfig, error) {
	var serverPublicKey []byte
	var chunkSize int

	untypedServerPublicKey, ok := args["serverPublicKey"]
	if !ok {
		return nil, errors.New("replicant transport clientConfig  missing serverPublicKey")
	}

	switch untypedServerPublicKey.(type) {
	case string:
		sequenceString, icError := interconv.ParseString(untypedServerPublicKey)
		if icError != nil {
			log.Errorf("could not parse clientConfig serverPublicKey string")
		}
		var byteErr error
		serverPublicKey, byteErr = hex.DecodeString(sequenceString)
		if byteErr != nil {
			log.Errorf("could not parse clientConfig serverPublicKey string bytes")
		}
	default:
		return nil, errors.New("unsupported type for replicant clientConfig serverPublicKey option")
	}

	untypedChunkSize, ok := args["chunkSize"]
	if !ok {
		return nil, errors.New("replicant transport missing clientConfig chunkSize argument")
	}

	switch untypedChunkSize.(type) {
	case float64:
		var icError error
		chunkSize, icError = interconv.ParseInt(untypedChunkSize)
		if icError != nil {
			log.Errorf("could not parse clientConfig chunkSize")
		}
	default:
		return nil, errors.New("unsupported type for clientConfig chunkSize option")
	}

	silverPolishClientConfig := polish.SilverPolishClientConfig{
		ServerPublicKey: serverPublicKey,
		ChunkSize:       chunkSize,
	}

	return &silverPolishClientConfig, nil
}

func parseServerConfig(args map[string]interface{}) (*polish.SilverPolishServerConfig, error) {
	var serverPublicKey []byte
	var serverPrivateKey []byte
	var chunkSize int

	untypedServerPublicKey, ok := args["serverPublicKey"]
	if !ok {
		return nil, errors.New("replicant transport serverConfig missing serverPublicKey")
	}

	switch untypedServerPublicKey.(type) {
	case string:
		sequenceString, icError := interconv.ParseString(untypedServerPublicKey)
		if icError != nil {
			log.Errorf("could not parse serverConfig serverPublicKey string")
		}
		var byteErr error
		serverPublicKey, byteErr = hex.DecodeString(sequenceString)
		if byteErr != nil {
			log.Errorf("could not parse serverConfig serverPublicKey string bytes")
		}
	default:
		return nil, errors.New("unsupported type for serverConfig serverPublicKey option")
	}

	untypedServerPrivateKey, ok := args["serverPublicKey"]
	if !ok {
		return nil, errors.New("replicant transport serverConfig missing serverPublicKey")
	}

	switch untypedServerPrivateKey.(type) {
	case string:
		serverPrivateKeyString, icError := interconv.ParseString(untypedServerPrivateKey)
		if icError != nil {
			log.Errorf("could not parse serverConfig serverPrivateKey string")
		}
		var byteErr error
		serverPrivateKey, byteErr = hex.DecodeString(serverPrivateKeyString)
		if byteErr != nil {
			log.Errorf("could not parse serverConfig serverPrivateKey string bytes")
		}
	default:
		return nil, errors.New("unsupported type for replicant serverConfig serverPrivateKey option")
	}

	untypedChunkSize, ok := args["chunkSize"]
	if !ok {
		return nil, errors.New("replicant transport serverConfig missing chunkSize argument")
	}

	switch untypedChunkSize.(type) {
	case float64:
		var icError error
		chunkSize, icError = interconv.ParseInt(untypedChunkSize)
		if icError != nil {
			log.Errorf("could not parse serverConfig chunkSize")
		}
	default:
		return nil, errors.New("unsupported type for replicant Silver Polish serverConfig option")
	}

	silverPolishServerConfig := polish.SilverPolishServerConfig{
		ServerPublicKey:  serverPublicKey,
		ServerPrivateKey: serverPrivateKey,
		ChunkSize:        chunkSize,
	}

	return &silverPolishServerConfig, nil
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

func ParseArgsOptimizer(args map[string]interface{}, dialer proxy.Dialer) (*Optimizer.Client, error) {
	var transports []Optimizer.Transport
	var strategy Optimizer.Strategy

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

	//start by parsing the address
	untypedAddress, ok := otc["address"]
	if !ok {
		return nil, errors.New("missing address in transport parser")
	}

	switch untypedAddress.(type) {

	case string:
		var icerr error
		address, icerr = interconv.ParseString(untypedAddress)
		if icerr != nil {
			return nil, icerr
		}

	default:
		return nil, errors.New("unsupported type for optimizer address option")
	}
	//now to parse the name
	untypedName, ok2 := otc["name"]
	if !ok2 {
		return nil, errors.New("missing name in transport parser")
	}

	switch untypedName.(type) {

	case string:
		var icerr error
		name, icerr = interconv.ParseString(untypedName)
		if icerr != nil {
			return nil, icerr
		}

	default:
		return nil, errors.New("unsupported type for optimizer name option")
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

	switch name {
	case "shadow":
		shadowTransport, parseErr := ParseArgsShadow(config, address, dialer)
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
		replicantTransport, parseErr := ParseArgsReplicant(config, address, dialer)
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
