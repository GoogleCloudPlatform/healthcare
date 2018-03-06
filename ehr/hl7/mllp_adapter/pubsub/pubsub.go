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

// Package pubsub handles notifications of messages that should be sent back to
// the partner.
package pubsub

import (
	"fmt"

	log "github.com/golang/glog"
	"golang.org/x/net/context"
	"mllp_adapter/monitoring"
	"cloud.google.com/go/pubsub"
	"google.golang.org/api/option"

	goauth2 "golang.org/x/oauth2/google"
)

const (
	fetchErrorMetric = "pubsub-messages-fetch-error"
	sendErrorMetric  = "pubsub-messages-send-error"
	processedMetric  = "pubsub-messages-processed"
	ignoredMetric    = "pubsub-messages-ignored"
)

type message interface {
	Ack()
	Data() []byte
	Attrs() map[string]string
}

type fetcher interface {
	Get(string) ([]byte, error)
}

type sender interface {
	Send([]byte) ([]byte, error)
}

type messageWrapper struct {
	msg *pubsub.Message
}

func (m *messageWrapper) Ack() {
	m.msg.Ack()
}

func (m *messageWrapper) Data() []byte {
	return m.msg.Data
}

func (m *messageWrapper) Attrs() map[string]string {
	return m.msg.Attributes
}

func handleMessage(m message, metrics *monitoring.Client, f fetcher, s sender) {
	metrics.Inc(processedMetric)

	// Ignore messages that are not meant to be published.
	if publishValue, ok := m.Attrs()["publish"]; !ok || publishValue != "true" {
		metrics.Inc(ignoredMetric)
		return
	}

	msgName := string(m.Data())
	msg, err := f.Get(msgName)
	if err != nil {
		log.Warningf("Error fetching message %v: %v", msgName, err)
		metrics.Inc(fetchErrorMetric)
		return
	}
	if _, err := s.Send(msg); err != nil {
		log.Warningf("Error sending message %v: %v", msgName, err)
		metrics.Inc(sendErrorMetric)
		return
	}

	log.Infof("Sent message %v", msgName)
	m.Ack()
}

func initMetrics(metrics *monitoring.Client) {
	metrics.NewInt64(fetchErrorMetric)
	metrics.NewInt64(sendErrorMetric)
	metrics.NewInt64(processedMetric)
	metrics.NewInt64(ignoredMetric)
}

// Listen listens for notifications from a pubsub subscription, uses the ids
// in the messages to fetch content with the HL7 API, then sends the message
// to the partner over MLLP.
func Listen(ctx context.Context, mt *monitoring.Client, projectID string, topic string, fetcher fetcher, sender sender) error {
	ts, err := goauth2.DefaultTokenSource(ctx)
	if err != nil {
		return fmt.Errorf("getting default token source: %v", err)
	}
	client, err := pubsub.NewClient(ctx, projectID, option.WithTokenSource(ts))
	if err != nil {
		return fmt.Errorf("creating pubsub client: %v", err)
	}

	initMetrics(mt)
	return client.Subscription(topic).Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		handleMessage(&messageWrapper{msg: msg}, mt, fetcher, sender)
	})
}
