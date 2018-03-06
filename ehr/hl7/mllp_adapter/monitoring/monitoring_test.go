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

package monitoring

import (
	"testing"
	"time"

	"golang.org/x/net/context"

	tspb "github.com/golang/protobuf/ptypes"
	metricpb "google.golang.org/genproto/googleapis/api/metric"
	monitoredrespb "google.golang.org/genproto/googleapis/api/monitoredres"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
	"github.com/kylelemons/godebug/pretty"
	"github.com/googleapis/gax-go"
)

type fakeClient struct {
	req    []*monitoringpb.CreateTimeSeriesRequest
	cancel context.CancelFunc
	closed bool
}

func (c *fakeClient) CreateTimeSeries(ctx context.Context, req *monitoringpb.CreateTimeSeriesRequest, opts ...gax.CallOption) error {
	c.req = append(c.req, req)
	c.cancel()
	return nil
}

func (c *fakeClient) Close() error {
	c.closed = true
	return nil
}

func TestClient(t *testing.T) {
	timeNow = func() time.Time { return time.Date(2017, 4, 25, 15, 57, 46, 123456, time.UTC) }
	now, err := tspb.TimestampProto(timeNow())
	if err != nil {
		t.Fatalf("TimeStampProto(): %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	fake := &fakeClient{cancel: cancel}
	cl := &Client{
		client:    fake,
		projectID: "my-project-id",
		labels: map[string]string{
			"job":      "mllp_adapter",
			"instance": "instance1",
			"zone":     "zone1",
		},
		metrics: make(map[string]*metric),
	}
	cl.NewInt64("test-metric")
	cl.Inc("test-metric")
	cl.Inc("test-metric")
	cl.Inc("test-metric")

	want := &monitoringpb.CreateTimeSeriesRequest{
		Name: "projects/my-project-id",
		TimeSeries: []*monitoringpb.TimeSeries{
			{
				Metric: &metricpb.Metric{
					Type: "custom.googleapis.com/cloud/healthcare/mllp/test-metric",
					Labels: map[string]string{
						"job":      "mllp_adapter",
						"zone":     "zone1",
						"instance": "instance1",
					},
				},
				Resource: &monitoredrespb.MonitoredResource{
					Type: "global",
					Labels: map[string]string{
						"project_id": "my-project-id",
					},
				},
				MetricKind: metricpb.MetricDescriptor_CUMULATIVE,
				ValueType:  metricpb.MetricDescriptor_INT64,
				Points: []*monitoringpb.Point{{
					Interval: &monitoringpb.TimeInterval{
						StartTime: now,
						EndTime:   now,
					},
					Value: &monitoringpb.TypedValue{
						Value: &monitoringpb.TypedValue_Int64Value{
							Int64Value: 3,
						},
					},
				}},
			}},
	}

	if err := cl.StartExport(ctx, 1, 1); err != nil {
		t.Errorf("Run() returned an error: %v", err)
	}

	if !fake.closed {
		t.Errorf("Client not closed")
	}
	if len(fake.req) != 1 {
		t.Errorf("Expected 1 request, got %v", len(fake.req))
	}
	got := fake.req[0]
	if diff := pretty.Compare(want, got); diff != "" {
		t.Errorf("unexpected response, got: \n%v\n, want: \n%v\n, diff (-want +got): \n%v\n", got, want, diff)
	}
}
