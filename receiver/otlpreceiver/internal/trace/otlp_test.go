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

package trace

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

	traceSink := new(consumertest.TracesSink)

	port, doneFn := otlpReceiverOnGRPCServer(t, traceSink)
	defer doneFn()

	traceClient, traceClientDoneFn, err := makeTraceServiceClient(port)
	require.NoError(t, err, "Failed to create the TraceServiceClient: %v", err)
	defer traceClientDoneFn()

	// when

	req := testdata.GenerateTracesOneSpan()

	// Keep trace data to compare the test result against it
	// Clone needed because OTLP proto XXX_ fields are altered in the GRPC downstream
	traceData := req.Clone()

	resp, err := traceClient.Export(context.Background(), req)
	require.NoError(t, err, "Failed to export trace: %v", err)
	require.NotNil(t, resp, "The response is missing")

	// assert
	require.Len(t, traceSink.AllTraces(), 1)
	assert.EqualValues(t, traceData, traceSink.AllTraces()[0])
}

func TestExport_EmptyRequest(t *testing.T) {
	traceSink := new(consumertest.TracesSink)

	addr, doneFn := otlpReceiverOnGRPCServer(t, traceSink)
	defer doneFn()

	traceClient, traceClientDoneFn, err := makeTraceServiceClient(addr)
	require.NoError(t, err, "Failed to create the TraceServiceClient: %v", err)
	defer traceClientDoneFn()

	resp, err := traceClient.Export(context.Background(), pdata.NewTraces())
	assert.NoError(t, err, "Failed to export trace: %v", err)
	assert.NotNil(t, resp, "The response is missing")
}

func TestExport_ErrorConsumer(t *testing.T) {
	addr, doneFn := otlpReceiverOnGRPCServer(t, consumertest.NewErr(errors.New("my error")))
	defer doneFn()

	traceClient, traceClientDoneFn, err := makeTraceServiceClient(addr)
	require.NoError(t, err, "Failed to create the TraceServiceClient: %v", err)
	defer traceClientDoneFn()

	req := testdata.GenerateTracesOneSpan()
	resp, err := traceClient.Export(context.Background(), req)
	assert.EqualError(t, err, "rpc error: code = Unknown desc = my error")
	assert.Equal(t, otlpgrpc.TracesResponse{}, resp)
}

func makeTraceServiceClient(addr net.Addr) (otlpgrpc.TracesClient, func(), error) {
	cc, err := grpc.Dial(addr.String(), grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, nil, err
	}

	metricsClient := otlpgrpc.NewTracesClient(cc)

	doneFn := func() { _ = cc.Close() }
	return metricsClient, doneFn, nil
}

func otlpReceiverOnGRPCServer(t *testing.T, tc consumer.Traces) (net.Addr, func()) {
	ln, err := net.Listen("tcp", "localhost:")
	require.NoError(t, err, "Failed to find an available address to run the gRPC server: %v", err)

	doneFnList := []func(){func() { ln.Close() }}
	done := func() {
		for _, doneFn := range doneFnList {
			doneFn()
		}
	}

	r := New(config.NewIDWithName("otlp", "trace"), tc)
	require.NoError(t, err)

	// Now run it as a gRPC server
	srv := grpc.NewServer()
	otlpgrpc.RegisterTracesServer(srv, r)
	go func() {
		_ = srv.Serve(ln)
	}()

	return ln.Addr(), done
}
