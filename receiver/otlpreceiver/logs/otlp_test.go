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

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal"
	"go.opentelemetry.io/collector/internal/data"
	collectorlog "go.opentelemetry.io/collector/internal/data/protogen/collector/logs/v1"
	otlplog "go.opentelemetry.io/collector/internal/data/protogen/logs/v1"
	"go.opentelemetry.io/collector/obsreport"
)

var _ collectorlog.LogsServiceServer = (*Receiver)(nil)

func TestExport(t *testing.T) {
	// given

	logSink := new(consumertest.LogsSink)

	addr, doneFn := otlpReceiverOnGRPCServer(t, logSink)
	defer doneFn()

	traceClient, traceClientDoneFn, err := makeLogsServiceClient(addr)
	require.NoError(t, err, "Failed to create the TraceServiceClient: %v", err)
	defer traceClientDoneFn()

	// when

	unixnanos := uint64(12578940000000012345)
	traceID := [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1}
	spanID := [8]byte{8, 7, 6, 5, 4, 3, 2, 1}
	otlp := &collectorlog.ExportLogsServiceRequest{
		ResourceLogs: []*otlplog.ResourceLogs{
			{
				InstrumentationLibraryLogs: []*otlplog.InstrumentationLibraryLogs{
					{
						Logs: []*otlplog.LogRecord{
							{
								TraceId:      data.NewTraceID(traceID),
								SpanId:       data.NewSpanID(spanID),
								Name:         "operationB",
								TimeUnixNano: unixnanos,
							},
						},
					},
				},
			},
		},
	}

	// Keep log data to compare the test result against it
	// Clone needed because OTLP proto XXX_ fields are altered in the GRPC downstream
	ld := pdata.LogsFromInternalRep(internal.LogsFromOtlp(otlp)).Clone()

	resp, err := traceClient.Export(context.Background(), otlp)
	require.NoError(t, err, "Failed to export trace: %v", err)
	require.NotNil(t, resp, "The response is missing")

	// assert

	require.Equal(t, 1, len(logSink.AllLogs()), "unexpected length: %v", len(logSink.AllLogs()))

	assert.EqualValues(t, ld, logSink.AllLogs()[0])
}

func TestExport_EmptyRequest(t *testing.T) {
	logSink := new(consumertest.LogsSink)

	addr, doneFn := otlpReceiverOnGRPCServer(t, logSink)
	defer doneFn()

	logClient, logClientDoneFn, err := makeLogsServiceClient(addr)
	require.NoError(t, err, "Failed to create the TraceServiceClient: %v", err)
	defer logClientDoneFn()

	resp, err := logClient.Export(context.Background(), &collectorlog.ExportLogsServiceRequest{})
	assert.NoError(t, err, "Failed to export trace: %v", err)
	assert.NotNil(t, resp, "The response is missing")
}

func TestExport_ErrorConsumer(t *testing.T) {
	addr, doneFn := otlpReceiverOnGRPCServer(t, consumertest.NewErr(errors.New("my error")))
	defer doneFn()

	logClient, logClientDoneFn, err := makeLogsServiceClient(addr)
	require.NoError(t, err, "Failed to create the TraceServiceClient: %v", err)
	defer logClientDoneFn()

	req := &collectorlog.ExportLogsServiceRequest{
		ResourceLogs: []*otlplog.ResourceLogs{
			{
				InstrumentationLibraryLogs: []*otlplog.InstrumentationLibraryLogs{
					{
						Logs: []*otlplog.LogRecord{
							{
								Name: "operationB",
							},
						},
					},
				},
			},
		},
	}

	resp, err := logClient.Export(context.Background(), req)
	assert.EqualError(t, err, "rpc error: code = Unknown desc = my error")
	assert.Nil(t, resp)
}

func makeLogsServiceClient(addr net.Addr) (collectorlog.LogsServiceClient, func(), error) {
	cc, err := grpc.Dial(addr.String(), grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, nil, err
	}

	logClient := collectorlog.NewLogsServiceClient(cc)

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

	r := New(receiverTagValue, tc)
	require.NoError(t, err)

	// Now run it as a gRPC server
	srv := obsreport.GRPCServerWithObservabilityEnabled()
	collectorlog.RegisterLogsServiceServer(srv, r)
	go func() {
		_ = srv.Serve(ln)
	}()

	return ln.Addr(), done
}
