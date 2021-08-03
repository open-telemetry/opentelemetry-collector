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

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/model/otlpgrpc"
	"go.opentelemetry.io/collector/model/pdata"
)

func TestExport(t *testing.T) {
	// given

	metricSink := new(consumertest.MetricsSink)

	port, doneFn := otlpReceiverOnGRPCServer(t, metricSink)
	defer doneFn()

	metricsClient, metricsClientDoneFn, err := makeMetricsServiceClient(port)
	require.NoError(t, err, "Failed to create the MetricsServiceClient: %v", err)
	defer metricsClientDoneFn()

	// when

	req := testdata.GenerateMetricsOneMetric()

	// Keep metric data to compare the test result against it
	// Clone needed because OTLP proto XXX_ fields are altered in the GRPC downstream
	metricData := req.Clone()

	resp, err := metricsClient.Export(context.Background(), req)
	require.NoError(t, err, "Failed to export metrics: %v", err)
	require.NotNil(t, resp, "The response is missing")

	// assert

	mds := metricSink.AllMetrics()
	require.Len(t, mds, 1)
	assert.EqualValues(t, metricData, mds[0])
}

func TestExport_EmptyRequest(t *testing.T) {
	// given

	metricSink := new(consumertest.MetricsSink)

	addr, doneFn := otlpReceiverOnGRPCServer(t, metricSink)
	defer doneFn()

	metricsClient, metricsClientDoneFn, err := makeMetricsServiceClient(addr)
	require.NoError(t, err, "Failed to create the MetricsServiceClient: %v", err)
	defer metricsClientDoneFn()

	resp, err := metricsClient.Export(context.Background(), pdata.NewMetrics())
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

	req := testdata.GenerateMetricsOneMetric()

	resp, err := metricsClient.Export(context.Background(), req)
	assert.EqualError(t, err, "rpc error: code = Unknown desc = my error")
	assert.Equal(t, otlpgrpc.MetricsResponse{}, resp)
}

func makeMetricsServiceClient(addr net.Addr) (otlpgrpc.MetricsClient, func(), error) {
	cc, err := grpc.Dial(addr.String(), grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, nil, err
	}

	metricsClient := otlpgrpc.NewMetricsClient(cc)

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

	r := New(config.NewIDWithName("otlp", "metrics"), mc)
	// Now run it as a gRPC server
	srv := grpc.NewServer()
	otlpgrpc.RegisterMetricsServer(srv, r)
	go func() {
		_ = srv.Serve(ln)
	}()

	return ln.Addr(), done
}
