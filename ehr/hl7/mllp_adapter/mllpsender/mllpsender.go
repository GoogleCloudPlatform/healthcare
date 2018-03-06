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

// Package mllpsender sends HL7 messages via MLLP.
package mllpsender

import (
	"fmt"
	"net"

	log "github.com/golang/glog"
	"mllp_adapter/mllp"
	"mllp_adapter/monitoring"
)

const (
	sentMetric      = "mllpsender-messages-sent"
	ackErrorMetric  = "mllpsender-messages-ack-error"
	sendErrorMetric = "mllpsender-messages-send-error"
	dialErrorMetric = "mllpsender-connections-dial-error"
)

// MLLPSender represents an MLLP sender.
type MLLPSender struct {
	addr    string
	metrics *monitoring.Client
}

// NewSender creates a new MLLPSender.
func NewSender(addr string, metrics *monitoring.Client) *MLLPSender {
	metrics.NewInt64(sentMetric)
	metrics.NewInt64(ackErrorMetric)
	metrics.NewInt64(sendErrorMetric)
	metrics.NewInt64(dialErrorMetric)
	return &MLLPSender{addr: addr, metrics: metrics}
}

// Send sends an HL7 messages via MLLP.
func (m *MLLPSender) Send(msg []byte) ([]byte, error) {
	m.metrics.Inc(sentMetric)

	log.Infof("Dialing MLLP connection")

	conn, err := net.Dial("tcp", m.addr)
	if err != nil {
		m.metrics.Inc(dialErrorMetric)
		return nil, fmt.Errorf("dialing: %v", err)
	}
	defer func() {
		log.Infof("Closing MLLP connection")
		if err := conn.Close(); err != nil {
			log.Errorf("Closing connection: %v", err)
		}
	}()

	log.Infof("Sending MLLP message")

	if err := mllp.WriteMsg(conn, msg); err != nil {
		m.metrics.Inc(sendErrorMetric)
		return nil, fmt.Errorf("writing message: %v", err)
	}
	ack, err := mllp.ReadMsg(conn)
	if err != nil {
		m.metrics.Inc(ackErrorMetric)
		return nil, fmt.Errorf("reading ack: %v", err)
	}
	log.Infof("Received MLLP ack")
	return ack, nil
}
