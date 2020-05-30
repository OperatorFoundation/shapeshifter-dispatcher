package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	socks "github.com/OperatorFoundation/shapeshifter-dispatcher/common/socks5"
	"io/ioutil"
	"net"
	"testing"
	"time"
)

const (
	version                = 0x05
)

func TestSocksTCPDust(t *testing.T) {
	negotiateError := negotiateSocks("dustClient.json")
	if negotiateError != nil {
		t.Fail()
	}
}

func TestSocksTCPMeek(t *testing.T) {
	negotiateError := negotiateSocks("meek.json")
	if negotiateError != nil {
		t.Fail()
	}
}

func TestSocksTCPObfs2(t *testing.T) {
	negotiateError := negotiateSocks("")
	if negotiateError != nil {
		t.Fail()
	}
}

func TestSocksTCPObfs4(t *testing.T) {
	negotiateError := negotiateSocks("obfs4.json")
	if negotiateError != nil {
		t.Fail()
	}
}

func TestSocksTCPOptimizerFirst(t *testing.T) {
	negotiateError := negotiateSocks("OptimizerFirst.json")
	if negotiateError != nil {
		t.Fail()
	}
}

func TestSocksTCPOptimizerMinimizeDialDuration(t *testing.T) {
	negotiateError := negotiateSocks("OptimizerMinimizeDialDuration.json")
	if negotiateError != nil {
		t.Fail()
	}
}

func TestSocksTCPOptimizerRandom(t *testing.T) {
	negotiateError := negotiateSocks("OptimizerRandom.json")
	if negotiateError != nil {
		t.Fail()
	}
}

func TestSocksTCPOptimizerRotate(t *testing.T) {
	negotiateError := negotiateSocks("OptimizerRotate.json")
	if negotiateError != nil {
		t.Fail()
	}
}

func TestSocksTCPOptimizerTrack(t *testing.T) {
	negotiateError := negotiateSocks("OptimizerTrack.json")
	if negotiateError != nil {
		t.Fail()
	}
}

func TestSocksTCPReplicant(t *testing.T) {
	negotiateError := negotiateSocks("ReplicantClient1.json")
	if negotiateError != nil {
		t.Fail()
	}
}

func TestSocksTCPShadow(t *testing.T) {
	negotiateError := negotiateSocks("shadowClient.json")
	if negotiateError != nil {
		t.Fail()
	}
}

func negotiateSocks(jsonFile string) error {
	dialConn, dialError := net.Dial("tcp", "127.0.0.1:1443")
	if dialError != nil {
		return dialError
	}

	time.Sleep(1 * time.Second)

	var err error
	var method byte

	// VER = 05, NMETHODS = 01, METHODS = [09]
	//Method 9 is the json parameter block authentication
	_, writeError := dialConn.Write([]byte{0x05, 0x01, 0x09})
	if writeError != nil {
		return writeError
	}

	if method, err = NegotiateAuth(dialConn); err != nil {
		return err
	}

	if method != socks.AuthJsonParameterBlock {
		return errors.New("negotiateAuth(jsonParameterBlock) unexpected method")
	}
	jsonData, jsonErr := ioutil.ReadFile(jsonFile)
	if jsonErr != nil {
		return jsonErr
	}
	jsonDataLength := len(jsonData)
	jsonLengthBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(jsonLengthBytes, uint32(jsonDataLength))

	_, byteLengthWriteError := dialConn.Write(jsonLengthBytes)
	if byteLengthWriteError != nil {
		return byteLengthWriteError
	}

	_, byteWriteError := dialConn.Write(jsonData)
	if byteWriteError != nil {
		return byteWriteError
	}

	time.Sleep(100 * time.Millisecond)

	request := []byte{0x05, 0x01, 0x00, 0x01, 0x7F, 0x00, 0x00, 0x01, 0x0D, 0x05}
	_, writeReqErr := dialConn.Write(request)
	if writeReqErr != nil {
		return writeReqErr
	}

	reply := make([]byte, 10)
	_, readError := dialConn.Read(reply)
	if readError != nil {
		return readError
	}

	if reply[0] != 0x05 {
		return errors.New("incorrect socks version")
	}

	if reply[1] != 0x00 {
		println(reply[1])
		return errors.New("non-successful reply")
	}

	if reply[3] != 0x01 {
		return errors.New("expected IPV4 address")
	}

	_, writeErr := dialConn.Write([]byte("data"))
	if writeErr != nil {
		return writeErr
	}

	return nil
}

func NegotiateAuth(req net.Conn) (byte, error) {
	// The client sends a version identifier/selection message.
	//	uint8_t ver (0x05)
	//  uint8_t nmethods (>= 1).
	//  uint8_t methods[nmethods]

	var err error
	if err = readByteVerify(req, "version", version); err != nil {
		return 0, err
	}

	// Read the number of methods, and the methods.
	var methods = make([]byte, 1)
	if _, err = req.Read(methods); err != nil {
		return 0, err
	}
	method := methods[0]

	return method, nil
}

func readByteVerify(req net.Conn, descr string, expected byte) error {
	var b = make([]byte, 1)
	_, err := req.Read(b)
	if err != nil {
		return err
	}
	val := b[0]
	if val != expected {
		return fmt.Errorf("message field '%s' was 0x%02x (expected 0x%02x)", descr, val, expected)
	}
	return nil
}
