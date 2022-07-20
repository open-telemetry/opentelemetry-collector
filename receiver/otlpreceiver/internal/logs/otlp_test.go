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

package logs

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
)

func TestExport(t *testing.T) {
	logSink := new(consumertest.LogsSink)

	addr, doneFn := otlpReceiverOnGRPCServer(t, logSink)
	defer doneFn()

	traceClient, traceClientDoneFn, err := makeLogsServiceClient(addr)
	require.NoError(t, err, "Failed to create the TraceServiceClient: %v", err)
	defer traceClientDoneFn()

	ld := testdata.GenerateLogs(1)
	// Keep log data to compare the test result against it
	// Clone needed because OTLP proto XXX_ fields are altered in the GRPC downstream
	logData := ld.Clone()
	req := plogotlp.NewRequestFromLogs(ld)

	resp, err := traceClient.Export(context.Background(), req)
	require.NoError(t, err, "Failed to export trace: %v", err)
	require.NotNil(t, resp, "The response is missing")

	lds := logSink.AllLogs()
	require.Len(t, lds, 1)
	assert.EqualValues(t, logData, lds[0])
}

func TestExport_EmptyRequest(t *testing.T) {
	logSink := new(consumertest.LogsSink)

	addr, doneFn := otlpReceiverOnGRPCServer(t, logSink)
	defer doneFn()

	logClient, logClientDoneFn, err := makeLogsServiceClient(addr)
	require.NoError(t, err, "Failed to create the TraceServiceClient: %v", err)
	defer logClientDoneFn()

	resp, err := logClient.Export(context.Background(), plogotlp.NewRequest())
	assert.NoError(t, err, "Failed to export trace: %v", err)
	assert.NotNil(t, resp, "The response is missing")
}

func TestExport_ErrorConsumer(t *testing.T) {
	addr, doneFn := otlpReceiverOnGRPCServer(t, consumertest.NewErr(errors.New("my error")))
	defer doneFn()

	logClient, logClientDoneFn, err := makeLogsServiceClient(addr)
	require.NoError(t, err, "Failed to create the TraceServiceClient: %v", err)
	defer logClientDoneFn()

	ld := testdata.GenerateLogs(1)
	req := plogotlp.NewRequestFromLogs(ld)

	resp, err := logClient.Export(context.Background(), req)
	assert.EqualError(t, err, "rpc error: code = Unknown desc = my error")
	assert.Equal(t, plogotlp.Response{}, resp)
}

func makeLogsServiceClient(addr net.Addr) (plogotlp.Client, func(), error) {
	cc, err := grpc.Dial(addr.String(), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		return nil, nil, err
	}

	logClient := plogotlp.NewClient(cc)

	doneFn := func() { _ = cc.Close() }
	return logClient, doneFn, nil
}

func otlpReceiverOnGRPCServer(t *testing.T, tc consumer.Logs) (net.Addr, func()) {
	ln, err := net.Listen("tcp", "localhost:")
	require.NoError(t, err, "Failed to find an available address to run the gRPC server: %v", err)

	doneFnList := []func(){func() { ln.Close() }}
	done := func() {
		for _, doneFn := range doneFnList {
			doneFn()
		}
	}

	r := New(config.NewComponentIDWithName("otlp", "log"), tc, componenttest.NewNopReceiverCreateSettings())
	require.NoError(t, err)

	// Now run it as a gRPC server
	srv := grpc.NewServer()
	plogotlp.RegisterServer(srv, r)
	go func() {
		_ = srv.Serve(ln)
	}()

	return ln.Addr(), done
}
