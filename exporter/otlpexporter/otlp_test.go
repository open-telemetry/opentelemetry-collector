// Copyright 2019, OpenTelemetry Authors
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

package otlpexporter

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	otlptracecol "github.com/open-telemetry/opentelemetry-proto/gen/go/collector/trace/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/component/componenttest"
	"github.com/open-telemetry/opentelemetry-collector/config/configgrpc"
	"github.com/open-telemetry/opentelemetry-collector/internal/data/testdata"
	"github.com/open-telemetry/opentelemetry-collector/observability"
	"github.com/open-telemetry/opentelemetry-collector/testutils"
)

type MockReceiver struct {
	requestCount   int32
	totalSpanCount int32
	lastRequest    *otlptracecol.ExportTraceServiceRequest
}

func (r *MockReceiver) Export(
	ctx context.Context,
	req *otlptracecol.ExportTraceServiceRequest,
) (*otlptracecol.ExportTraceServiceResponse, error) {
	atomic.AddInt32(&r.requestCount, 1)
	spanCount := 0
	for _, rs := range req.ResourceSpans {
		for _, ils := range rs.InstrumentationLibrarySpans {
			spanCount += len(ils.Spans)
		}
	}
	atomic.AddInt32(&r.totalSpanCount, int32(spanCount))
	r.lastRequest = req
	return &otlptracecol.ExportTraceServiceResponse{}, nil
}

func otlpReceiverOnGRPCServer(t *testing.T) (r *MockReceiver, port int, done func()) {
	ln, err := net.Listen("tcp", "localhost:")
	require.NoError(t, err, "Failed to find an available address to run the gRPC server: %v", err)

	doneFnList := []func(){func() { ln.Close() }}
	done = func() {
		for _, doneFn := range doneFnList {
			doneFn()
		}
	}

	_, port, err = hostPortFromAddr(ln.Addr())
	if err != nil {
		done()
		t.Fatalf("Failed to parse host:port from listener address: %s error: %v", ln.Addr(), err)
	}

	r = &MockReceiver{}
	require.NoError(t, err, "Failed to create the Receiver: %v", err)

	// Now run it as a gRPC server
	srv := observability.GRPCServerWithObservabilityEnabled()
	otlptracecol.RegisterTraceServiceServer(srv, r)
	go func() {
		_ = srv.Serve(ln)
	}()

	return r, port, done
}

func hostPortFromAddr(addr net.Addr) (host string, port int, err error) {
	addrStr := addr.String()
	sepIndex := strings.LastIndex(addrStr, ":")
	if sepIndex < 0 {
		return "", -1, errors.New("failed to parse host:port")
	}
	host, portStr := addrStr[:sepIndex], addrStr[sepIndex+1:]
	port, err = strconv.Atoi(portStr)
	return host, port, err
}

func TestSendTraceData(t *testing.T) {
	// Start an OTLP-compatible receiver.
	rcv, port, done := otlpReceiverOnGRPCServer(t)
	defer done()

	// Start an OTLP exporter and point to the receiver.

	endpoint := fmt.Sprintf("localhost:%d", port)

	config := Config{
		GRPCSettings: configgrpc.GRPCSettings{
			Endpoint: endpoint,
		},
	}

	factory := &Factory{}
	creationParams := component.ExporterCreateParams{Logger: zap.NewNop()}
	exp, err := factory.CreateTraceExporter(context.Background(), creationParams, &config)
	assert.Nil(t, err)
	require.NotNil(t, exp)
	defer exp.Shutdown(context.Background())

	host := componenttest.NewNopHost()

	err = exp.Start(context.Background(), host)
	assert.NoError(t, err)

	// Ensure that initially there is no data in the receiver.
	assert.EqualValues(t, 0, rcv.requestCount)

	// Send empty trace.
	td := testdata.GenerateTraceDataEmpty()
	exp.ConsumeTrace(context.Background(), td)

	// Wait until it is received.
	testutils.WaitFor(t, func() bool {
		return atomic.LoadInt32(&rcv.requestCount) > 0
	}, "receive a request")

	// Ensure it was received empty.
	assert.EqualValues(t, 0, rcv.totalSpanCount)

	// A trace with 2 spans.
	td = testdata.GenerateTraceDataTwoSpansSameResource()

	expectedOTLPReq := &otlptracecol.ExportTraceServiceRequest{
		ResourceSpans: testdata.GenerateTraceOtlpSameResourceTwoSpans(),
	}

	err = exp.ConsumeTrace(context.Background(), td)
	assert.NoError(t, err)

	// Wait until it is received.
	testutils.WaitFor(t, func() bool {
		return atomic.LoadInt32(&rcv.requestCount) > 1
	}, "receive a request")

	// Verify received span.
	assert.EqualValues(t, 2, rcv.totalSpanCount)
	assert.EqualValues(t, expectedOTLPReq, rcv.lastRequest)
}
