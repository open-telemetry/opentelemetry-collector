// Copyright The OpenTelemetry Authors
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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdatautil"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/internal/data"
	collectormetrics "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/collector/metrics/v1"
	otlpcommon "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/common/v1"
	otlpmetrics "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/metrics/v1"
	otlpresource "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/resource/v1"
	"go.opentelemetry.io/collector/observability"
	"go.opentelemetry.io/collector/testutil"
)

var _ collectormetrics.MetricsServiceServer = (*Receiver)(nil)

func TestExport(t *testing.T) {
	// given

	metricSink := new(exportertest.SinkMetricsExporter)

	_, port, doneFn := otlpReceiverOnGRPCServer(t, metricSink)
	defer doneFn()

	metricsClient, metricsClientDoneFn, err := makeMetricsServiceClient(port)
	require.NoError(t, err, "Failed to create the MetricsServiceClient: %v", err)
	defer metricsClientDoneFn()

	// when

	unixnanos1 := uint64(12578940000000012345)
	unixnanos2 := uint64(12578940000000054321)

	resourceMetrics := []*otlpmetrics.ResourceMetrics{
		{
			Resource: &otlpresource.Resource{
				Attributes: []*otlpcommon.KeyValue{
					{
						Key:   "key1",
						Value: &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "value1"}},
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
								Type:        otlpmetrics.MetricDescriptor_MONOTONIC_INT64,
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
	}

	// Keep metric data to compare the test result against it
	// Clone needed because OTLP proto XXX_ fields are altered in the GRPC downstream
	metricData := data.MetricDataFromOtlp(resourceMetrics).Clone()

	req := &collectormetrics.ExportMetricsServiceRequest{
		ResourceMetrics: resourceMetrics,
	}

	resp, err := metricsClient.Export(context.Background(), req)
	require.NoError(t, err, "Failed to export metrics: %v", err)
	require.NotNil(t, resp, "The response is missing")

	// assert

	require.Equal(t, 1, len(metricSink.AllMetrics()),
		"unexpected length: %v", len(metricSink.AllMetrics()))

	assert.EqualValues(t, metricData, pdatautil.MetricsToInternalMetrics(metricSink.AllMetrics()[0]))
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

func otlpReceiverOnGRPCServer(t *testing.T, mc consumer.MetricsConsumer) (r *Receiver, port int, done func()) {
	ln, err := net.Listen("tcp", "localhost:")
	require.NoError(t, err, "Failed to find an available address to run the gRPC server: %v", err)

	doneFnList := []func(){func() { ln.Close() }}
	done = func() {
		for _, doneFn := range doneFnList {
			doneFn()
		}
	}

	_, port, err = testutil.HostPortFromAddr(ln.Addr())
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
