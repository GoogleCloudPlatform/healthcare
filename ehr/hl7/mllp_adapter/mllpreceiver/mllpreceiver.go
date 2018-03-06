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

// Package mllpreceiver receives HL7 messages over MLLP and forwards them to a sender.
package mllpreceiver

import (
	"fmt"
	"net"
	"strconv"
	"time"

	log "github.com/golang/glog"
	"mllp_adapter/mllp"
	"mllp_adapter/monitoring"
)

// The sender interface represents the destination to which HL7 messages are sent, one at a time.
type sender interface {
	Send([]byte) ([]byte, error)
}

// MLLPReceiver represents an MLLP receiver.
type MLLPReceiver struct {
	listener net.Listener
	sender   sender
	port     int
	metrics  *monitoring.Client

	// If non-nil, connClosed will receive a message every time a connection
	// is closed.  This is primarily useful for synchronizing tests.
	connClosed chan struct{}
}

const (
	reconnectsMetric     = "receiver-reconnects"
	readsMetric          = "receiver-reads"
	handleMessagesMetric = "receiver-handle-messages"
	writesMetric         = "receiver-writes"
)

// NewReceiver creates a new MLLP receiver.  If port is 0, an available port is
// chosen at random.
func NewReceiver(ip string, port int, sender sender, mt *monitoring.Client) (*MLLPReceiver, error) {
	localhost := net.JoinHostPort(ip, strconv.Itoa(port))
	l, err := net.Listen("tcp", localhost)
	if err != nil {
		return nil, fmt.Errorf("Listen: %v", err)
	}

	tcpAddr, ok := l.Addr().(*net.TCPAddr)
	if !ok {
		return nil, fmt.Errorf("casting %v to TCPAddr: %v", l.Addr(), err)
	}
	mt.NewInt64(reconnectsMetric)
	mt.NewInt64(readsMetric)
	mt.NewInt64(handleMessagesMetric)
	mt.NewInt64(writesMetric)
	return &MLLPReceiver{listener: l, sender: sender, metrics: mt, port: tcpAddr.Port}, nil
}

// Run starts listening for incoming TCP connections. Only returns in case of an error.
func (m *MLLPReceiver) Run() error {
	defer func() {
		if err := m.listener.Close(); err != nil {
			log.Errorf("Closing listener: %v", err)
		}
	}()
	for {
		conn, err := m.listener.(*net.TCPListener).AcceptTCP()
		if err != nil {
			return fmt.Errorf("AcceptTCP: %v", err)
		}
		m.metrics.Inc(reconnectsMetric)
		go m.handleConnection(conn)
	}
}

// handleConnection handles a single TCP connection.
func (m *MLLPReceiver) handleConnection(conn *net.TCPConn) {
	defer func() {
		if err := conn.Close(); err != nil {
			log.Errorf("Closing connection: %v", err)
		}
		if m.connClosed != nil {
			m.connClosed <- struct{}{}
		}
	}()
	// Cloud VPC resets connections that are idle for 10 minutes (see
	// https://cloud.google.com/compute/docs/networks-and-firewalls), so we
	// send a keep alive message every 3 minutes to keep that from
	// happening.
	conn.SetKeepAlive(true)
	conn.SetKeepAlivePeriod(3 * time.Minute)

	log.Infof("Accepted connection from %v", conn.RemoteAddr())
	defer log.Infof("Closed connection from %v", conn.RemoteAddr())
	for {
		msg, err := mllp.ReadMsg(conn)
		if err != nil {
			log.Errorf("reading message: %v", err)
			return
		}
		m.metrics.Inc(readsMetric)
		ack, err := m.handleMessage(msg)
		if err != nil {
			log.Errorf("handleMessage: %v", err)
			return
		}
		m.metrics.Inc(handleMessagesMetric)
		if err := mllp.WriteMsg(conn, ack); err != nil {
			log.Errorf("writing ack: %v", err)
			return
		}
		m.metrics.Inc(writesMetric)
	}
}

func (m *MLLPReceiver) handleMessage(msg []byte) ([]byte, error) {
	ack, err := m.sender.Send(msg)
	if err != nil {
		return nil, fmt.Errorf("Send: %v", err)
	}
	return ack, nil
}
