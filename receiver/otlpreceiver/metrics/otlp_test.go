// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metrics

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal"
	collectormetrics "go.opentelemetry.io/collector/internal/data/protogen/collector/metrics/v1"
	otlpcommon "go.opentelemetry.io/collector/internal/data/protogen/common/v1"
	otlpmetrics "go.opentelemetry.io/collector/internal/data/protogen/metrics/v1"
	"go.opentelemetry.io/collector/obsreport"
)

var _ collectormetrics.MetricsServiceServer = (*Receiver)(nil)

func TestExport(t *testing.T) {
	// given

	metricSink := new(consumertest.MetricsSink)

	port, doneFn := otlpReceiverOnGRPCServer(t, metricSink)
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
				InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
					{
						Metrics: []*otlpmetrics.Metric{
							{
								Name:        "mymetric",
								Description: "My metric",
								Unit:        "ms",
								Data: &otlpmetrics.Metric_IntSum{
									IntSum: &otlpmetrics.IntSum{
										IsMonotonic:            true,
										AggregationTemporality: otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
										DataPoints: []*otlpmetrics.IntDataPoint{
											{
												Labels: []otlpcommon.StringKeyValue{
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
												Labels: []otlpcommon.StringKeyValue{
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
			},
		},
	}

	// Keep metric data to compare the test result against it
	// Clone needed because OTLP proto XXX_ fields are altered in the GRPC downstream
	metricData := pdata.MetricsFromInternalRep(internal.MetricsFromOtlp(req)).Clone()

	resp, err := metricsClient.Export(context.Background(), req)
	require.NoError(t, err, "Failed to export metrics: %v", err)
	require.NotNil(t, resp, "The response is missing")

	// assert

	require.Equal(t, 1, len(metricSink.AllMetrics()),
		"unexpected length: %v", len(metricSink.AllMetrics()))

	assert.EqualValues(t, metricData, metricSink.AllMetrics()[0])
}

func TestExport_EmptyRequest(t *testing.T) {
	// given

	metricSink := new(consumertest.MetricsSink)

	addr, doneFn := otlpReceiverOnGRPCServer(t, metricSink)
	defer doneFn()

	metricsClient, metricsClientDoneFn, err := makeMetricsServiceClient(addr)
	require.NoError(t, err, "Failed to create the MetricsServiceClient: %v", err)
	defer metricsClientDoneFn()

	resp, err := metricsClient.Export(context.Background(), &collectormetrics.ExportMetricsServiceRequest{})
	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestExport_ErrorConsumer(t *testing.T) {
	// given

	addr, doneFn := otlpReceiverOnGRPCServer(t, consumertest.NewErr(errors.New("my error")))
	defer doneFn()

	metricsClient, metricsClientDoneFn, err := makeMetricsServiceClient(addr)
	require.NoError(t, err, "Failed to create the MetricsServiceClient: %v", err)
	defer metricsClientDoneFn()

	req := &collectormetrics.ExportMetricsServiceRequest{ResourceMetrics: []*otlpmetrics.ResourceMetrics{
		{
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					Metrics: []*otlpmetrics.Metric{
						{
							Name:        "mymetric",
							Description: "My metric",
							Unit:        "ms",
							Data: &otlpmetrics.Metric_IntSum{
								IntSum: &otlpmetrics.IntSum{
									IsMonotonic:            true,
									AggregationTemporality: otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
									DataPoints: []*otlpmetrics.IntDataPoint{
										{
											Value: 123,
										},
										{
											Value: 456,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}}
	resp, err := metricsClient.Export(context.Background(), req)
	assert.EqualError(t, err, "rpc error: code = Unknown desc = my error")
	assert.Nil(t, resp)
}

func makeMetricsServiceClient(addr net.Addr) (collectormetrics.MetricsServiceClient, func(), error) {
	cc, err := grpc.Dial(addr.String(), grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, nil, err
	}

	metricsClient := collectormetrics.NewMetricsServiceClient(cc)

	doneFn := func() { _ = cc.Close() }
	return metricsClient, doneFn, nil
}

func otlpReceiverOnGRPCServer(t *testing.T, mc consumer.Metrics) (net.Addr, func()) {
	ln, err := net.Listen("tcp", "localhost:")
	require.NoError(t, err, "Failed to find an available address to run the gRPC server: %v", err)

	doneFnList := []func(){func() { ln.Close() }}
	done := func() {
		for _, doneFn := range doneFnList {
			doneFn()
		}
	}

	r := New(receiverTagValue, mc)
	// Now run it as a gRPC server
	srv := obsreport.GRPCServerWithObservabilityEnabled()
	collectormetrics.RegisterMetricsServiceServer(srv, r)
	go func() {
		_ = srv.Serve(ln)
	}()

	return ln.Addr(), done
}
