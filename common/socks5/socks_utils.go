package socks5

import (
	"bufio"
	"bytes"
	"encoding/hex"
)

// TestReadWriter is a bytes.Buffer backed io.ReadWriter used for testing.  The
// Read and Write routines are to be used by the component being tested.  Data
// can be written to and read back via the WriteHex and ReadHex routines.
type TestReadWriter struct {
	readBuf  bytes.Buffer
	writeBuf bytes.Buffer
}

func (c *TestReadWriter) Read(buf []byte) (n int, err error) {
	return c.readBuf.Read(buf)
}

func (c *TestReadWriter) Write(buf []byte) (n int, err error) {
	return c.writeBuf.Write(buf)
}

func (c *TestReadWriter) WriteHex(str string) (n int, err error) {
	var buf []byte
	if buf, err = hex.DecodeString(str); err != nil {
		return
	}
	return c.readBuf.Write(buf)
}

func (c *TestReadWriter) ReadHex() string {
	return hex.EncodeToString(c.writeBuf.Bytes())
}

func (c *TestReadWriter) toBufio() *bufio.ReadWriter {
	return bufio.NewReadWriter(bufio.NewReader(c), bufio.NewWriter(c))
}

func (c *TestReadWriter) ToRequest() *Request {
	req := new(Request)
	req.rw = c.toBufio()
	return req
}

func (c *TestReadWriter) reset(req *Request) {
	c.readBuf.Reset()
	c.writeBuf.Reset()
	req.rw = c.toBufio()
}
