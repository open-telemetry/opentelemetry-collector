// Copyright 2020, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metrics

import (
	"context"
	"fmt"
	"net"
	"testing"

	occommon "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	ocmetrics "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	ocresource "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"github.com/golang/protobuf/ptypes/timestamp"
	collectormetrics "github.com/open-telemetry/opentelemetry-proto/gen/go/collector/metrics/v1"
	otlpcommon "github.com/open-telemetry/opentelemetry-proto/gen/go/common/v1"
	otlpmetrics "github.com/open-telemetry/opentelemetry-proto/gen/go/metrics/v1"
	otlpresource "github.com/open-telemetry/opentelemetry-proto/gen/go/resource/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/internal"
	"github.com/open-telemetry/opentelemetry-collector/internal/data"
	"github.com/open-telemetry/opentelemetry-collector/observability"
	"github.com/open-telemetry/opentelemetry-collector/receiver/receivertest"
	"github.com/open-telemetry/opentelemetry-collector/testutils"
)

var _ collectormetrics.MetricsServiceServer = (*Receiver)(nil)

func TestExport(t *testing.T) {
	// given

	metricSink := receivertest.NewMetricAppender()

	_, port, doneFn := otlpReceiverOnGRPCServer(t, metricSink)
	defer doneFn()

	metricsClient, metricsClientDoneFn, err := makeMetricsServiceClient(port)
	require.NoError(t, err, "Failed to create the MetricsServiceClient: %v", err)
	defer metricsClientDoneFn()

	// when

	unixnanos1 := uint64(12578940000000012345)
	unixnanos2 := uint64(12578940000000054321)

	req := &collectormetrics.ExportMetricsServiceRequest{
		ResourceMetrics: []*otlpmetrics.ResourceMetrics{
			{
				Resource: &otlpresource.Resource{
					Attributes: []*otlpcommon.AttributeKeyValue{
						{
							Key:         "key1",
							StringValue: "value1",
						},
					},
				},
				InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
					{
						InstrumentationLibrary: &otlpcommon.InstrumentationLibrary{
							Name:    "name1",
							Version: "version1",
						},
						Metrics: []*otlpmetrics.Metric{
							{
								MetricDescriptor: &otlpmetrics.MetricDescriptor{
									Name:        "mymetric",
									Description: "My metric",
									Unit:        "ms",
									Type:        otlpmetrics.MetricDescriptor_COUNTER_INT64,
								},
								Int64DataPoints: []*otlpmetrics.Int64DataPoint{
									{
										Labels: []*otlpcommon.StringKeyValue{
											{
												Key:   "key1",
												Value: "value1",
											},
										},
										StartTimeUnixNano: unixnanos1,
										TimeUnixNano:      unixnanos2,
										Value:             123,
									},
									{
										Labels: []*otlpcommon.StringKeyValue{
											{
												Key:   "key2",
												Value: "value2",
											},
										},
										StartTimeUnixNano: unixnanos1,
										TimeUnixNano:      unixnanos2,
										Value:             456,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	resp, err := metricsClient.Export(context.Background(), req)
	require.NoError(t, err, "Failed to export metrics: %v", err)
	require.NotNil(t, resp, "The response is missing")

	// assert

	require.Equal(t, 1, len(metricSink.MetricsDataList),
		"unexpected length: %v", len(metricSink.MetricsDataList))

	metricsData := consumerdata.MetricsData{
		Node: &occommon.Node{},
		Resource: &ocresource.Resource{
			Labels: map[string]string{
				"key1": "value1",
			},
		},
		Metrics: []*ocmetrics.Metric{
			{
				MetricDescriptor: &ocmetrics.MetricDescriptor{
					Name:        "mymetric",
					Description: "My metric",
					Unit:        "ms",
					Type:        ocmetrics.MetricDescriptor_CUMULATIVE_INT64,
					LabelKeys: []*ocmetrics.LabelKey{
						{Key: "key1"},
						{Key: "key2"},
					},
				},
				Timeseries: []*ocmetrics.TimeSeries{
					{
						StartTimestamp: unixnanoToTimestamp(unixnanos1),
						LabelValues: []*ocmetrics.LabelValue{
							{
								// key1
								Value:    "value1",
								HasValue: true,
							},
							{
								// key2
								HasValue: false,
							},
						},
						Points: []*ocmetrics.Point{
							{
								Timestamp: unixnanoToTimestamp(unixnanos2),
								Value: &ocmetrics.Point_Int64Value{
									Int64Value: 123,
								},
							},
						},
					},
					{
						StartTimestamp: unixnanoToTimestamp(unixnanos1),
						LabelValues: []*ocmetrics.LabelValue{
							{
								// key1
								HasValue: false,
							},
							{
								// key2
								Value:    "value2",
								HasValue: true,
							},
						},
						Points: []*ocmetrics.Point{
							{
								Timestamp: unixnanoToTimestamp(unixnanos2),
								Value: &ocmetrics.Point_Int64Value{
									Int64Value: 456,
								},
							},
						},
					},
				},
			},
		},
	}

	assert.EqualValues(t, metricsData, metricSink.MetricsDataList[0])
}

func makeMetricsServiceClient(port int) (collectormetrics.MetricsServiceClient, func(), error) {
	addr := fmt.Sprintf(":%d", port)
	cc, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, nil, err
	}

	metricsClient := collectormetrics.NewMetricsServiceClient(cc)

	doneFn := func() { _ = cc.Close() }
	return metricsClient, doneFn, nil
}

func otlpReceiverOnGRPCServer(t *testing.T, mc consumer.MetricsConsumerOld) (r *Receiver, port int, done func()) {
	ln, err := net.Listen("tcp", "localhost:")
	require.NoError(t, err, "Failed to find an available address to run the gRPC server: %v", err)

	doneFnList := []func(){func() { ln.Close() }}
	done = func() {
		for _, doneFn := range doneFnList {
			doneFn()
		}
	}

	_, port, err = testutils.HostPortFromAddr(ln.Addr())
	if err != nil {
		done()
		t.Fatalf("Failed to parse host:port from listener address: %s error: %v", ln.Addr(), err)
	}

	r, err = New(receiverTagValue, mc)
	require.NoError(t, err, "Failed to create the Receiver: %v", err)

	// Now run it as a gRPC server
	srv := observability.GRPCServerWithObservabilityEnabled()
	collectormetrics.RegisterMetricsServiceServer(srv, r)
	go func() {
		_ = srv.Serve(ln)
	}()

	return r, port, done
}

func unixnanoToTimestamp(u uint64) *timestamp.Timestamp {
	return internal.UnixNanoToTimestamp(data.TimestampUnixNano(u))
}
