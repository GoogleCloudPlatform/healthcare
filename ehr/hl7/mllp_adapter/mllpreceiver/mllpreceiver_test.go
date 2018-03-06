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

package mllpreceiver

import (
	"net"
	"reflect"
	"strconv"
	"testing"

	"mllp_adapter/mllp"
	"mllp_adapter/monitoring"
)

var (
	cannedMsg  = []byte("abcd")
	cannedAck  = []byte("ack")
	wrappedMsg = []byte("\x0babcd\x1c\x0d")
)

type fakeSender struct {
	msgs [][]byte
}

func (s *fakeSender) Send(msg []byte) ([]byte, error) {
	s.msgs = append(s.msgs, msg)
	return cannedAck, nil
}

func setUp(t *testing.T) (*fakeSender, *MLLPReceiver) {
	s := &fakeSender{}
	mt := monitoring.NewClient()
	r, err := NewReceiver("0.0.0.0", 0, s, mt)
	// We want to be notified of closed connections.
	r.connClosed = make(chan struct{})
	if err != nil {
		t.Fatalf("NewReceiver: %v", err)
	}
	go r.Run()
	return s, r
}

type testCase struct {
	name         string
	connections  []connection
	expectedMsgs [][]byte
}

type connection struct {
	input        []byte
	expectedAcks [][]byte
}

func TestValidMessages(t *testing.T) {
	testCases := []testCase{
		testCase{
			"1 encapsulated message",
			[]connection{
				connection{
					wrappedMsg,
					[][]byte{cannedAck},
				},
			},
			[][]byte{cannedMsg},
		},
		testCase{
			"3 encapsulated messages, sent over separate connections",
			[]connection{
				connection{
					wrappedMsg,
					[][]byte{cannedAck},
				},
				connection{
					wrappedMsg,
					[][]byte{cannedAck},
				},
				connection{
					wrappedMsg,
					[][]byte{cannedAck},
				},
			},
			[][]byte{cannedMsg, cannedMsg, cannedMsg},
		},
		testCase{
			"2 encapsulated messages, 1 unencapsulated message (ignored and not sent), sent over separate connections",
			[]connection{
				connection{
					wrappedMsg,
					[][]byte{cannedAck},
				},
				connection{
					cannedMsg,
					nil,
				},
				connection{
					wrappedMsg,
					[][]byte{cannedAck},
				},
			},
			[][]byte{cannedMsg, cannedMsg},
		},
		testCase{
			"encapsulated message, unencapsulated message, encapsulated message, sent over a single connection, unencapsulated message and everything after is ignored",
			[]connection{
				connection{
					append(append(wrappedMsg, cannedMsg...), wrappedMsg...),
					[][]byte{cannedAck},
				},
			},
			[][]byte{cannedMsg},
		},
		testCase{
			"garbage (ignored)",
			[]connection{
				connection{
					[]byte("garbage"),
					nil,
				},
			},
			nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s, r := setUp(t)
			for _, c := range tc.connections {
				conn := dial(t, r.port)
				conn.Write(c.input)
				for _, expectedAck := range c.expectedAcks {
					ack := receiveAck(t, conn)
					if !reflect.DeepEqual(ack, expectedAck) {
						t.Errorf("Expected ack %v but got %v", expectedAck, ack)
					}
				}
				conn.Close()
			}

			waitForConnections(r, len(tc.connections))

			if !reflect.DeepEqual(tc.expectedMsgs, s.msgs) {
				t.Errorf("Expected message %v but got %v", tc.expectedMsgs, s.msgs)
			}
		})
	}
}

func Test3SimultanousConnections(t *testing.T) {
	s, r := setUp(t)
	c1 := dial(t, r.port)
	c2 := dial(t, r.port)
	c3 := dial(t, r.port)
	mllp.WriteMsg(c3, cannedMsg)
	c2.Write(cannedMsg) // Unencapsulated message, should be ignored
	mllp.WriteMsg(c1, cannedMsg)
	c1.Close()
	c2.Close()
	c3.Close()

	waitForConnections(r, 3)
	expected := [][]byte{cannedMsg, cannedMsg}
	if !reflect.DeepEqual(expected, s.msgs) {
		t.Fatalf("Messages differ: expected %v but got %v", expected, s.msgs)
	}
	if r.metrics.Value(reconnectsMetric) != 3 {
		t.Fatalf("Expected 3 reconnects but got %v", r.metrics.Value(reconnectsMetric))
	}
}

func TestMessageStats(t *testing.T) {
	s, r := setUp(t)
	c := dial(t, r.port)
	mllp.WriteMsg(c, cannedMsg)
	if err := c.Close(); err != nil {
		t.Fatalf("Failure closing connection: %v", err)
	}

	waitForConnections(r, 1)
	expected := [][]byte{cannedMsg}
	if !reflect.DeepEqual(expected, s.msgs) {
		t.Fatalf("Messages differ: expected %v but got %v", expected, s.msgs)
	}

	metrics := []string{readsMetric, handleMessagesMetric, writesMetric}
	for _, m := range metrics {
		if r.metrics.Value(m) != 1 {
			t.Errorf("Expected %v = 1 but got %v", m, r.metrics.Value(m))
		}
	}
}

func dial(t *testing.T, port int) net.Conn {
	c, err := net.Dial("tcp", net.JoinHostPort("localhost", strconv.Itoa(port)))
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	return c
}

func receiveAck(t *testing.T, c net.Conn) []byte {
	ack, err := mllp.ReadMsg(c)
	if err != nil {
		t.Fatalf("Reading ack: %v", err)
	}
	return ack
}

func waitForConnections(r *MLLPReceiver, count int) {
	for i := 0; i < count; i++ {
		<-r.connClosed
	}
}
