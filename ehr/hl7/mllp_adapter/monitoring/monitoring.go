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

// Package monitoring implements the functionality to export timeseries data to
// the Cloud Monitoring service.
package monitoring

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	log "github.com/golang/glog"
	"golang.org/x/net/context"
	timestamp "github.com/golang/protobuf/ptypes/timestamp"
	"github.com/googleapis/gax-go"
	"cloud.google.com/go/compute/metadata"
	"cloud.google.com/go/monitoring/apiv3"
	"google.golang.org/api/option"

	tspb "github.com/golang/protobuf/ptypes"
	metricpb "google.golang.org/genproto/googleapis/api/metric"
	monitoredrespb "google.golang.org/genproto/googleapis/api/monitoredres"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
	goauth2 "golang.org/x/oauth2/google"
)

// timeNow is used for testing.
var timeNow = time.Now

type metricClient interface {
	CreateTimeSeries(ctx context.Context, req *monitoringpb.CreateTimeSeriesRequest, opts ...gax.CallOption) error
	Close() error
}

// NewClient returns a client that can be used for creating, incrementing, and
// retrieving metrics but not for exporting them.
func NewClient() *Client {
	return &Client{metrics: make(map[string]*metric), labels: make(map[string]string)}
}

// ConfigureExport prepares a Client for exporting metrics. It fetches metadata
// about the GCP environment and fails if not running on GCE or GKE.
func ConfigureExport(ctx context.Context, cl *Client) error {
	if !metadata.OnGCE() {
		return fmt.Errorf("not running on GCE - metrics cannot be exported")
	}
	ts, err := goauth2.DefaultTokenSource(ctx)
	if err != nil {
		return fmt.Errorf("getting default token source: %v", err)
	}
	cl.client, err = monitoring.NewMetricClient(ctx, option.WithTokenSource(ts))
	if err != nil {
		return err
	}
	cl.projectID, err = metadata.ProjectID()
	if err != nil {
		return err
	}
	cl.labels["zone"], err = metadata.Zone()
	if err != nil {
		return err
	}
	cl.labels["instance"], err = metadata.InstanceName()
	if err != nil {
		return err
	}
	cl.labels["job"] = "mllp_adapter"
	return nil
}

// Client exports metrics to the Cloud Monitoring Service.
type Client struct {
	client    metricClient
	projectID string
	labels    map[string]string

	// mu guards metrics.  The other fields are immutable
	mu      sync.RWMutex
	metrics map[string]*metric
}

// Inc increments a metric or does nothing if the client is nil.
func (m *Client) Inc(name string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metrics[name].value++
}

// Value gets the value of a metric or returns 0 if the client is nil.
func (m *Client) Value(name string) int64 {
	if m == nil {
		return 0
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.metrics[name].value
}

// NewInt64 creates a new cumulative int64 metric with the given name or does
// nothing if the client is nil.  If the same name is used multiple times, only
// the last returned Metric is exported.
func (m *Client) NewInt64(name string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	startTime, err := tspb.TimestampProto(timeNow())
	if err != nil {
		log.Fatalf("Invalid timestamp: %v", err)
		return
	}
	m.metrics[name] = &metric{startTime: startTime}
}

// Run sends the metrics to the monitoring service roughly once a minute.
func (m *Client) StartExport(ctx context.Context, minStartDelay int, startDelayInterval int) error {
	// Use a variable delay so that processes starting at the same time
	// don't publish data at the same time. The initial delay is between
	// minStartDelay and minStartDelay+startDelayInterval seconds. Then, the
	// delay becomes 1 minute.
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	delay := time.Duration(minStartDelay+r.Intn(startDelayInterval)) * time.Second
	for {
		select {
		case <-time.After(delay):
			delay = time.Minute
		case <-ctx.Done():
			return m.client.Close()
		}
		req := m.makeCreateTimeSeriesRequest()
		if req == nil {
			continue
		}
		if err := m.client.CreateTimeSeries(ctx, req); err != nil {
			log.Errorf("CreateTimeSeries failed: %v", err)
		}
	}
}

func (m *Client) makeCreateTimeSeriesRequest() *monitoringpb.CreateTimeSeriesRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.metrics) == 0 {
		return nil
	}

	now, err := tspb.TimestampProto(timeNow())
	if err != nil {
		log.Errorf("Invalid timestamp: %v", err)
		return nil
	}
	req := &monitoringpb.CreateTimeSeriesRequest{
		Name: monitoring.MetricProjectPath(m.projectID),
	}
	for k, v := range m.metrics {
		dataPoint := &monitoringpb.Point{
			Interval: &monitoringpb.TimeInterval{
				StartTime: v.startTime,
				EndTime:   now,
			},
			Value: &monitoringpb.TypedValue{
				Value: &monitoringpb.TypedValue_Int64Value{
					Int64Value: v.value,
				},
			},
		}
		ts := &monitoringpb.TimeSeries{
			Metric: &metricpb.Metric{
				Type:   "custom.googleapis.com/cloud/healthcare/mllp/" + k,
				Labels: m.labels,
			},
			Resource: &monitoredrespb.MonitoredResource{
				Type: "global",
				Labels: map[string]string{
					"project_id": m.projectID,
				},
			},
			MetricKind: metricpb.MetricDescriptor_CUMULATIVE,
			ValueType:  metricpb.MetricDescriptor_INT64,
			Points: []*monitoringpb.Point{
				dataPoint,
			},
		}
		req.TimeSeries = append(req.TimeSeries, ts)
	}
	return req
}

// metric is a numeric value that is exported to the monitoring service.
type metric struct {
	startTime *timestamp.Timestamp
	value     int64
}
