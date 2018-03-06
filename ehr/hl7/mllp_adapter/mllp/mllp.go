// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package mllp contains functionality for encoding and decoding HL7 messages for transmission using the MLLP protocol.
// See here for the specification:
// http://www.hl7.org/documentcenter/public_temp_670395EE-1C23-BA17-0CD218684D5B3C71/wg/inm/mllp_transport_specification.PDF
package mllp

import (
	"bufio"
	"fmt"
	"io"
)

const (
	startBlock = '\x0b'
	endBlock   = '\x1c'
	cr         = '\x0d'
)

func encapsulate(in []byte) []byte {
	out := make([]byte, len(in)+3)
	out[0] = startBlock
	for i, b := range in {
		out[i+1] = b
	}
	out[len(out)-2] = endBlock
	out[len(out)-1] = cr
	return out
}

// WriteMsg wraps an HL7 message in the start block, end block, and carriage return bytes
// required for MLLP transmission and then writes the wrapped message to writer.
func WriteMsg(writer io.Writer, msg []byte) error {
	if _, err := writer.Write(encapsulate(msg)); err != nil {
		return fmt.Errorf("writing message: %v", err)
	}
	return nil
}

func checkByte(msg []byte, pos int, expected byte) error {
	if msg[pos] != expected {
		return fmt.Errorf("invalid message %v, expected %v at position %v but got %v",
			msg, expected, pos, msg[pos])
	}
	return nil
}

func decapsulate(msg []byte) ([]byte, error) {
	if len(msg) < 3 {
		return nil, fmt.Errorf("short message, length %v", len(msg))
	}
	if err := checkByte(msg, 0, startBlock); err != nil {
		return nil, err
	}
	if err := checkByte(msg, len(msg)-2, endBlock); err != nil {
		return nil, err
	}
	if err := checkByte(msg, len(msg)-1, cr); err != nil {
		return nil, err
	}
	return msg[1 : len(msg)-2], nil
}

// ReadMsg reads a message from reader and removes the start block, end block, and carriage return bytes.
func ReadMsg(reader io.Reader) ([]byte, error) {
	r := bufio.NewReader(reader)
	// Read everything up to the endBlock byte.
	rawMsg, err := r.ReadBytes(endBlock)
	if err != nil {
		return nil, fmt.Errorf("reading message: %v", err)
	}
	// Read one more byte for the carriage return.
	lastByte, err := r.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("reading last byte: %v", err)
	}
	msg, err := decapsulate(append(rawMsg, lastByte))
	if err != nil {
		return nil, fmt.Errorf("decapsulating message: %v", err)
	}
	return msg, nil
}
