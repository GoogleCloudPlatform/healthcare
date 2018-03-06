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

package mllp

import (
	"bytes"
	"testing"
)

func TestOK(t *testing.T) {
	testCases := []struct {
		name string
		raw  []byte
		mllp []byte
	}{
		{
			"empty",
			[]byte(""),
			[]byte("\x0b\x1c\x0d"),
		},
		{
			"non-empty",
			[]byte("msg"),
			[]byte("\x0bmsg\x1c\x0d"),
		},
		{
			"embedded carriage return",
			[]byte("msg\x0dmsg"),
			[]byte("\x0bmsg\x0dmsg\x1c\x0d"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			enc := &bytes.Buffer{}
			if err := WriteMsg(enc, tc.raw); err != nil {
				t.Errorf("Unexpected error writing message %v: %v", tc.raw, err)
			}
			if !bytes.Equal(enc.Bytes(), tc.mllp) {
				t.Errorf("Writing message %v: expected %v but got %v", tc.raw, tc.mllp, enc.Bytes())
			}

			dec, err := ReadMsg(bytes.NewBuffer(tc.mllp))
			if err != nil {
				t.Errorf("Unexpected error reading message %v: %v", tc.mllp, err)
			}
			if !bytes.Equal(dec, tc.raw) {
				t.Errorf("Reading message %v: expected %v but got %v", tc.mllp, tc.raw, dec)
			}
		})
	}
}

func TestError(t *testing.T) {
	testCases := []struct {
		name string
		msg  []byte
	}{
		{"missing start block", []byte("msg\x1c\x0d")},
		{"missing end block", []byte("\x0bmsg\x0d")},
		{"missing carriage return", []byte("\x0bmsg\x1c")},
		{"too short", []byte("\x0b\x0d")},
		{"empty", []byte("")},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ReadMsg(bytes.NewBuffer(tc.msg)); err == nil {
				t.Errorf("Expected error for message %v", tc.msg)
			}
		})
	}
}
