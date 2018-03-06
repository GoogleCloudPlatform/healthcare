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

package mllpsender

import (
	"bytes"
	"net"
	"strconv"
	"testing"

	"mllp_adapter/mllp"
	"mllp_adapter/monitoring"
	"mllp_adapter/testingutil"
)

var (
	cannedMsg = []byte("msg")
	cannedAck = []byte("ack")
)

func setUp() (*net.TCPListener, *MLLPSender, *monitoring.Client) {
	l, _ := net.Listen("tcp", "0.0.0.0:0")
	metrics := monitoring.NewClient()
	sender := NewSender(net.JoinHostPort("localhost", strconv.Itoa(l.Addr().(*net.TCPAddr).Port)), metrics)
	return l.(*net.TCPListener), sender, metrics
}

func accept(t *testing.T, listener *net.TCPListener) *net.TCPConn {
	conn, err := listener.AcceptTCP()
	if err != nil {
		t.Errorf("Unexpected error accepting connection: %v", err)
	}
	return conn
}

func TestOK(t *testing.T) {
	listener, sender, metrics := setUp()
	received := make(chan []byte)
	go func() {
		conn := accept(t, listener)
		msg, _ := mllp.ReadMsg(conn)
		mllp.WriteMsg(conn, cannedAck)
		received <- msg
		conn.Close()
	}()
	ack, err := sender.Send(cannedMsg)
	if err != nil {
		t.Errorf("Unexpected send error: %v", err)
	}
	if !bytes.Equal(cannedAck, ack) {
		t.Errorf("Expected ack %v, got %v", cannedAck, ack)
	}

	msgReceived := <-received
	if !bytes.Equal(cannedMsg, msgReceived) {
		t.Errorf("Expected msg %v, got %v", cannedMsg, msgReceived)
	}
	testingutil.CheckMetrics(t, metrics, map[string]int64{sentMetric: 1, ackErrorMetric: 0, dialErrorMetric: 0})
}

func TestDialError(t *testing.T) {
	listener, sender, metrics := setUp()
	listener.Close()
	if _, err := sender.Send(cannedMsg); err == nil {
		t.Errorf("Expected send error")
	}
	testingutil.CheckMetrics(t, metrics, map[string]int64{sentMetric: 1, ackErrorMetric: 0, dialErrorMetric: 1})
}

func TestSendError(t *testing.T) {
	listener, sender, metrics := setUp()
	go func() {
		conn := accept(t, listener)
		conn.Close()
	}()
	if _, err := sender.Send(cannedMsg); err == nil {
		t.Errorf("Expected send error")
	}
	testingutil.CheckMetrics(t, metrics, map[string]int64{sentMetric: 1, ackErrorMetric: 1, dialErrorMetric: 0})
}

func TestRecoverAfterSendError(t *testing.T) {
	listener, sender, metrics := setUp()
	go func() {
		conn := accept(t, listener)
		conn.Close()
		conn = accept(t, listener)
		mllp.ReadMsg(conn)
		mllp.WriteMsg(conn, cannedAck)
		conn.Close()
	}()
	if _, err := sender.Send(cannedMsg); err == nil {
		t.Errorf("Expected send error")
	}
	if _, err := sender.Send(cannedMsg); err != nil {
		t.Errorf("Unexpected send error: %v", err)
	}
	testingutil.CheckMetrics(t, metrics, map[string]int64{sentMetric: 2, ackErrorMetric: 1, dialErrorMetric: 0})
}

func TestTwoMessages(t *testing.T) {
	listener, sender, metrics := setUp()
	go func() {
		conn := accept(t, listener)
		mllp.ReadMsg(conn)
		mllp.WriteMsg(conn, cannedAck)
		conn.Close()
		conn = accept(t, listener)
		mllp.ReadMsg(conn)
		mllp.WriteMsg(conn, cannedAck)
		conn.Close()
	}()
	if _, err := sender.Send(cannedMsg); err != nil {
		t.Errorf("Unexpected send error: %v", err)
	}
	if _, err := sender.Send(cannedMsg); err != nil {
		t.Errorf("Unexpected send error: %v", err)
	}
	testingutil.CheckMetrics(t, metrics, map[string]int64{sentMetric: 2, ackErrorMetric: 0, dialErrorMetric: 0})
}
