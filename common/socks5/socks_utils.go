/*
 * Copyright (c) 2015, Yawning Angel <yawning at torproject dot org>
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
