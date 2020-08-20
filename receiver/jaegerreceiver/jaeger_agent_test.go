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

package jaegerreceiver

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"contrib.go.opencensus.io/exporter/jaeger"
	"github.com/jaegertracing/jaeger/proto-gen/api_v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/testutil"
)

const jaegerAgent = "jaeger_agent_test"

func TestJaegerAgentUDP_ThriftCompact_6831(t *testing.T) {
	port := 6831
	addrForClient := fmt.Sprintf(":%d", port)
	testJaegerAgent(t, addrForClient, &configuration{
		AgentCompactThriftPort: port,
	})
}

func TestJaegerAgentUDP_ThriftCompact_InvalidPort(t *testing.T) {
	port := 999999

	config := &configuration{
		AgentCompactThriftPort: port,
	}
	params := component.ReceiverCreateParams{Logger: zap.NewNop()}
	jr, err := newJaegerReceiver(jaegerAgent, config, nil, params)
	assert.NoError(t, err, "Failed to create new Jaeger Receiver")

	err = jr.Start(context.Background(), componenttest.NewNopHost())
	assert.Error(t, err, "should not have been able to startTraceReception")

	jr.Shutdown(context.Background())
}

func TestJaegerAgentUDP_ThriftBinary_6832(t *testing.T) {
	t.Skipf("Unfortunately due to Jaeger internal versioning, OpenCensus-Go's Thrift seems to conflict with ours")

	port := 6832
	addrForClient := fmt.Sprintf(":%d", port)
	testJaegerAgent(t, addrForClient, &configuration{
		AgentBinaryThriftPort: port,
	})
}

func TestJaegerAgentUDP_ThriftBinary_PortInUse(t *testing.T) {
	// This test confirms that the thrift binary port is opened correctly.  This is all we can test at the moment.  See above.
	port := testutil.GetAvailablePort(t)

	config := &configuration{
		AgentBinaryThriftPort: int(port),
	}
	params := component.ReceiverCreateParams{Logger: zap.NewNop()}
	jr, err := newJaegerReceiver(jaegerAgent, config, nil, params)
	assert.NoError(t, err, "Failed to create new Jaeger Receiver")

	err = jr.(*jReceiver).startAgent(componenttest.NewNopHost())
	assert.NoError(t, err, "Start failed")
	defer jr.Shutdown(context.Background())

	l, err := net.Listen("udp", fmt.Sprintf("localhost:%d", port))
	assert.Error(t, err, "should not have been able to listen to the port")

	if l != nil {
		l.Close()
	}
}

func TestJaegerAgentUDP_ThriftBinary_InvalidPort(t *testing.T) {
	port := 999999

	config := &configuration{
		AgentBinaryThriftPort: port,
	}
	params := component.ReceiverCreateParams{Logger: zap.NewNop()}
	jr, err := newJaegerReceiver(jaegerAgent, config, nil, params)
	assert.NoError(t, err, "Failed to create new Jaeger Receiver")

	err = jr.Start(context.Background(), componenttest.NewNopHost())
	assert.Error(t, err, "should not have been able to startTraceReception")

	jr.Shutdown(context.Background())
}

func initializeGRPCTestServer(t *testing.T, beforeServe func(server *grpc.Server), opts ...grpc.ServerOption) (*grpc.Server, net.Addr) {
	server := grpc.NewServer(opts...)
	lis, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	beforeServe(server)
	go func() {
		err := server.Serve(lis)
		require.NoError(t, err)
	}()
	return server, lis.Addr()
}

type mockSamplingHandler struct {
}

func (*mockSamplingHandler) GetSamplingStrategy(context.Context, *api_v2.SamplingStrategyParameters) (*api_v2.SamplingStrategyResponse, error) {
	return &api_v2.SamplingStrategyResponse{StrategyType: api_v2.SamplingStrategyType_PROBABILISTIC}, nil
}

func TestJaegerHTTP(t *testing.T) {
	s, addr := initializeGRPCTestServer(t, func(s *grpc.Server) {
		api_v2.RegisterSamplingManagerServer(s, &mockSamplingHandler{})
	})
	defer s.GracefulStop()

	port := testutil.GetAvailablePort(t)
	config := &configuration{
		AgentHTTPPort: int(port),
		RemoteSamplingClientSettings: configgrpc.GRPCClientSettings{
			Endpoint: addr.String(),
			TLSSetting: configtls.TLSClientSetting{
				Insecure: true,
			},
		},
	}
	params := component.ReceiverCreateParams{Logger: zap.NewNop()}
	jr, err := newJaegerReceiver(jaegerAgent, config, nil, params)
	assert.NoError(t, err, "Failed to create new Jaeger Receiver")
	defer jr.Shutdown(context.Background())

	err = jr.Start(context.Background(), componenttest.NewNopHost())
	assert.NoError(t, err, "Start failed")

	// allow http server to start
	err = testutil.WaitForPort(t, port)
	assert.NoError(t, err, "WaitForPort failed")

	testURL := fmt.Sprintf("http://localhost:%d/sampling?service=test", port)
	resp, err := http.Get(testURL)
	assert.NoError(t, err, "should not have failed to make request")
	if resp != nil {
		assert.Equal(t, 200, resp.StatusCode, "should have returned 200")
	}

	testURL = fmt.Sprintf("http://localhost:%d/sampling?service=test", port)
	resp, err = http.Get(testURL)
	assert.NoError(t, err, "should not have failed to make request")
	if resp != nil {
		assert.Equal(t, 200, resp.StatusCode, "should have returned 200")
	}

	testURL = fmt.Sprintf("http://localhost:%d/baggageRestrictions?service=test", port)
	resp, err = http.Get(testURL)
	assert.NoError(t, err, "should not have failed to make request")
	if resp != nil {
		assert.Equal(t, 200, resp.StatusCode, "should have returned 200")
	}
}

func testJaegerAgent(t *testing.T, agentEndpoint string, receiverConfig *configuration) {
	// 1. Create the Jaeger receiver aka "server"
	sink := new(exportertest.SinkTraceExporter)
	params := component.ReceiverCreateParams{Logger: zap.NewNop()}
	jr, err := newJaegerReceiver(jaegerAgent, receiverConfig, sink, params)
	assert.NoError(t, err, "Failed to create new Jaeger Receiver")
	defer jr.Shutdown(context.Background())

	err = jr.Start(context.Background(), componenttest.NewNopHost())
	assert.NoError(t, err, "Start failed")

	now := time.Unix(1542158650, 536343000).UTC()
	nowPlus10min := now.Add(10 * time.Minute)
	nowPlus10min2sec := now.Add(10 * time.Minute).Add(2 * time.Second)

	// 2. Then with a "live application", send spans to the Jaeger exporter.
	jexp, err := jaeger.NewExporter(jaeger.Options{
		AgentEndpoint: agentEndpoint,
		ServiceName:   "TestingAgentUDP",
		Process: jaeger.Process{
			ServiceName: "issaTest",
			Tags: []jaeger.Tag{
				jaeger.BoolTag("bool", true),
				jaeger.StringTag("string", "yes"),
				jaeger.Int64Tag("int64", 1e7),
			},
		},
	})
	assert.NoError(t, err, "Failed to create the Jaeger OpenCensus exporter for the live application")

	// 3. Now finally send some spans
	spandata := traceFixture(now, nowPlus10min, nowPlus10min2sec)

	for _, sd := range spandata {
		jexp.ExportSpan(sd)
	}
	jexp.Flush()

	// Simulate and account for network latency but also the reception process on the server.
	<-time.After(500 * time.Millisecond)

	for i := 0; i < 10; i++ {
		jexp.Flush()
		<-time.After(60 * time.Millisecond)
	}

	gotTraces := sink.AllTraces()
	assert.Equal(t, 1, len(gotTraces))

	want := expectedTraceData(now, nowPlus10min, nowPlus10min2sec)

	assert.EqualValues(t, want, gotTraces[0])
}
