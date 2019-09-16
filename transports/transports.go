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
	"errors"
	"fmt"
	"github.com/OperatorFoundation/shapeshifter-transports/transports/Dust"
	"github.com/OperatorFoundation/shapeshifter-transports/transports/Optimizer"
	replicant "github.com/OperatorFoundation/shapeshifter-transports/transports/Replicant"
	"github.com/OperatorFoundation/shapeshifter-transports/transports/meeklite"
	"github.com/OperatorFoundation/shapeshifter-transports/transports/obfs4"
	"github.com/OperatorFoundation/shapeshifter-transports/transports/shadow"
	"github.com/mufti1/interconv/package"
	gourl "net/url"
	"strconv"
)

// Transports returns the list of registered transport protocols.
func Transports() []string {
	return []string{"obfs2", "shadow", "obfs4", "Optimizer"}
}

func ParseArgsObfs4(args map[string]interface{}, target string) (*obfs4.Transport, error) {
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
		return nil, errors.New("Unsupported type for obfs4 cert option")
	}

	untypedIatMode, ok2 := args["iatMode"]
	if !ok2 {
		return nil, errors.New("obfs4 transport missing iatMode argument")
	}

	switch untypedCert.(type) {
	case string:
		iatModeStr, icerr := interconv.ParseString(untypedIatMode)
		if icerr != nil {
			return nil, icerr
		}
		iatModeInt, scerr := strconv.Atoi(iatModeStr)
		if scerr != nil {
			return nil, errors.New("obfs4 transport bad iatMode value")
		}
		switch iatModeInt {
		case 0:
			iatMode = iatModeInt
		case 1:
			iatMode = iatModeInt
		default:
			return nil, errors.New("Unsupported value for obfs4 iatMode option")
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
			return nil, errors.New("Unsupported value for obfs4 iatMode option")
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
			return nil, errors.New("Unsupported value for obfs4 iatMode option")
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
		return nil, errors.New("Unsupported type for obfs4 iatMode option")
	}

	transport := obfs4.Transport{
		CertString: cert,
		IatMode:    iatMode,
		Address:    target,
	}

	return &transport, nil
}

func ParseArgsShadow(args map[string]interface{}, target string) (*shadow.Transport, error) {
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
	}

	return &transport, nil
}

func ParseArgsDust(args map[string]interface{}, target string) (*Dust.Transport, error) {
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
	}

	return &transport, nil
}

func ParseArgsReplicant(args map[string]interface{}, target string) (*replicant.Transport, error) {
	var conf string
	fmt.Println(conf)
	untypedConfig, ok := args["config"]
	if !ok {
		return nil, errors.New("replicant transport missing config argument")
	}

	switch untypedConfig.(type) {
	case string:
		var icerr error
		conf, icerr = interconv.ParseString(untypedConfig)
		if icerr != nil {
			return nil, icerr
		}
	default:
		return nil, errors.New("unsupported type for replicant config option")
	}

	transport := replicant.Transport{
		Config:  replicant.Config{},
		Address: target,
	}

	return &transport, nil
}

func ParseArgsMeeklite(args map[string]interface{}, target string) (*meeklite.Transport, error) {

	var url *gourl.URL
	var front string

	untypedUrl, ok := args["url"]
	if !ok {
		return nil, errors.New("meeklite transport missing url argument")
	}

	switch untypedUrl.(type) {
	case gourl.URL:
		var icerr error
		url, icerr = interconv.ParseString(untypedUrl)
		if icerr != nil {
			return nil, icerr
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
	}

	return &transport, nil
}

func ParseArgsOptimizer(args map[string]interface{}, target string) (*Optimizer.OptimizerTransport, error) {
	var transports string
	var strategy string

	untypedTransports, ok := args["transports"]
	if !ok {
		return nil, errors.New("optimizer transport missing transports argument")
	}

	switch untypedTransports.(type) {
	case string:
		var icerr error
		transports, icerr = interconv.ParseString(untypedTransports)
		if icerr != nil {
			return nil, icerr
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
		var icerr error
		strategy, icerr = interconv.ParseString(untypedStrategy)
		if icerr != nil {
			return nil, icerr
		}
	default:
		return nil, errors.New("unsupported type for optimizer strategy option")
	}

	transport := Optimizer.OptimizerTransport{
		Transports: transports,
		Strategy:   strategy,
	}

	return &transport, nil
}
