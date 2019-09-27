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

package jaegerreceiver

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	agentapp "github.com/jaegertracing/jaeger/cmd/agent/app"
	"github.com/jaegertracing/jaeger/cmd/agent/app/configmanager"
	"github.com/jaegertracing/jaeger/cmd/agent/app/reporter"
	"github.com/jaegertracing/jaeger/cmd/collector/app"
	"github.com/jaegertracing/jaeger/proto-gen/api_v2"
	"github.com/jaegertracing/jaeger/thrift-gen/baggage"
	"github.com/jaegertracing/jaeger/thrift-gen/jaeger"
	"github.com/jaegertracing/jaeger/thrift-gen/sampling"
	"github.com/jaegertracing/jaeger/thrift-gen/zipkincore"
	"github.com/uber/jaeger-lib/metrics"
	"github.com/uber/tchannel-go"
	"github.com/uber/tchannel-go/thrift"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/observability"
	"github.com/open-telemetry/opentelemetry-collector/oterr"
	"github.com/open-telemetry/opentelemetry-collector/receiver"
	jaegertranslator "github.com/open-telemetry/opentelemetry-collector/translator/trace/jaeger"
)

// Configuration defines the behavior and the ports that
// the Jaeger receiver will use.
type Configuration struct {
	CollectorThriftPort  int
	CollectorHTTPPort    int
	CollectorGRPCPort    int
	CollectorGRPCOptions []grpc.ServerOption

	AgentPort              int
	AgentCompactThriftPort int
	AgentBinaryThriftPort  int
}

// Receiver type is used to receive spans that were originally intended to be sent to Jaeger.
// This receiver is basically a Jaeger collector.
type jReceiver struct {
	// mu protects the fields of this type
	mu sync.Mutex

	nextConsumer consumer.TraceConsumer

	startOnce sync.Once
	stopOnce  sync.Once

	config *Configuration

	agent *agentapp.Agent

	grpc            *grpc.Server
	tchanServer     *jTchannelReceiver
	collectorServer *http.Server

	defaultAgentCtx context.Context
}

type jTchannelReceiver struct {
	nextConsumer consumer.TraceConsumer

	tchannel *tchannel.Channel
}

const (
	// As per https://www.jaegertracing.io/docs/1.13/deployment/
	// By default, the port used by jaeger-agent to send spans in model.proto format
	defaultGRPCPort = 14250
	// By default, the port used by jaeger-agent to send spans in jaeger.thrift format
	defaultTChannelPort = 14267
	// By default, can accept spans directly from clients in jaeger.thrift format over binary thrift protocol
	defaultCollectorHTTPPort = 14268

	// As per https://www.jaegertracing.io/docs/1.7/deployment/#agent
	// 5775	UDP accept zipkin.thrift over compact thrift protocol
	// 6831	UDP accept jaeger.thrift over compact thrift protocol
	// 6832	UDP accept jaeger.thrift over binary thrift protocol
	defaultCompactThriftUDPPort = 6831
	defaultBinaryThriftUDPPort  = 6832

	traceSource string = "Jaeger"
)

// New creates a TraceReceiver that receives traffic as a collector with both Thrift and HTTP transports.
func New(ctx context.Context, config *Configuration, nextConsumer consumer.TraceConsumer) (receiver.TraceReceiver, error) {
	return &jReceiver{
		config:          config,
		defaultAgentCtx: observability.ContextWithReceiverName(context.Background(), "jaeger-agent"),
		nextConsumer:    nextConsumer,
		tchanServer: &jTchannelReceiver{
			nextConsumer: nextConsumer,
		},
	}, nil
}

var _ receiver.TraceReceiver = (*jReceiver)(nil)

func (jr *jReceiver) collectorAddr() string {
	var port int
	if jr.config != nil {
		port = jr.config.CollectorHTTPPort
	}
	if port <= 0 {
		port = defaultCollectorHTTPPort
	}
	return fmt.Sprintf(":%d", port)
}

const defaultAgentPort = 5778

func (jr *jReceiver) agentAddress() string {
	var port int
	if jr.config != nil {
		port = jr.config.AgentPort
	}
	if port <= 0 {
		port = defaultAgentPort
	}
	return fmt.Sprintf(":%d", port)
}

// TODO https://github.com/open-telemetry/opentelemetry-collector/issues/267
//	Remove ThriftTChannel support.
func (jr *jReceiver) tchannelAddr() string {
	var port int
	if jr.config != nil {
		port = jr.config.CollectorThriftPort
	}
	if port <= 0 {
		port = defaultTChannelPort
	}
	return fmt.Sprintf(":%d", port)
}

func (jr *jReceiver) grpcAddr() string {
	var port int
	if jr.config != nil {
		port = jr.config.CollectorGRPCPort
	}
	if port <= 0 {
		port = defaultGRPCPort
	}
	return fmt.Sprintf(":%d", port)
}

func (jr *jReceiver) agentCompactThriftAddr() string {
	var port int
	if jr.config != nil {
		port = jr.config.AgentCompactThriftPort
	}
	if port <= 0 {
		port = defaultCompactThriftUDPPort
	}
	return fmt.Sprintf(":%d", port)
}

func (jr *jReceiver) agentBinaryThriftAddr() string {
	var port int
	if jr.config != nil {
		port = jr.config.AgentBinaryThriftPort
	}
	if port <= 0 {
		port = defaultBinaryThriftUDPPort
	}
	return fmt.Sprintf(":%d", port)
}

func (jr *jReceiver) TraceSource() string {
	return traceSource
}

func (jr *jReceiver) StartTraceReception(host receiver.Host) error {
	jr.mu.Lock()
	defer jr.mu.Unlock()

	var err = oterr.ErrAlreadyStarted
	jr.startOnce.Do(func() {
		if err = jr.startAgent(host); err != nil && err != oterr.ErrAlreadyStarted {
			jr.stopTraceReceptionLocked()
			return
		}

		if err = jr.startCollector(host); err != nil && err != oterr.ErrAlreadyStarted {
			jr.stopTraceReceptionLocked()
			return
		}

		err = nil
	})
	return err
}

func (jr *jReceiver) StopTraceReception() error {
	jr.mu.Lock()
	defer jr.mu.Unlock()

	return jr.stopTraceReceptionLocked()
}

func (jr *jReceiver) stopTraceReceptionLocked() error {
	var err = oterr.ErrAlreadyStopped
	jr.stopOnce.Do(func() {
		var errs []error

		if jr.agent != nil {
			jr.agent.Stop()
			jr.agent = nil
		}

		if jr.collectorServer != nil {
			if cerr := jr.collectorServer.Close(); cerr != nil {
				errs = append(errs, cerr)
			}
			jr.collectorServer = nil
		}
		if jr.tchanServer.tchannel != nil {
			jr.tchanServer.tchannel.Close()
			jr.tchanServer.tchannel = nil
		}
		if jr.grpc != nil {
			jr.grpc.Stop()
			jr.grpc = nil
		}
		if len(errs) == 0 {
			err = nil
			return
		}
		// Otherwise combine all these errors
		buf := new(bytes.Buffer)
		for _, err := range errs {
			fmt.Fprintf(buf, "%s\n", err.Error())
		}
		err = errors.New(buf.String())
	})

	return err
}

const collectorReceiverTagValue = "jaeger-collector"
const tchannelCollectorReceiverTagValue = "jaeger-tchannel-collector"

func consumeTraceData(ctx context.Context, batches []*jaeger.Batch, consumer consumer.TraceConsumer) ([]*jaeger.BatchSubmitResponse, error) {
	jbsr := make([]*jaeger.BatchSubmitResponse, 0, len(batches))

	for _, batch := range batches {
		td, err := jaegertranslator.ThriftBatchToOCProto(batch)
		// TODO: (@odeke-em) add this error for Jaeger observability
		ok := false

		if err == nil {
			ok = true
			td.SourceFormat = "jaeger"
			consumer.ConsumeTraceData(ctx, td)
			// We MUST unconditionally record metrics from this reception.
			observability.RecordMetricsForTraceReceiver(ctx, len(batch.Spans), len(batch.Spans)-len(td.Spans))
		}

		jbsr = append(jbsr, &jaeger.BatchSubmitResponse{
			Ok: ok,
		})
	}

	return jbsr, nil
}

func (jr *jReceiver) SubmitBatches(batches []*jaeger.Batch, options app.SubmitBatchOptions) ([]*jaeger.BatchSubmitResponse, error) {
	ctx := context.Background()
	ctxWithReceiverName := observability.ContextWithReceiverName(ctx, collectorReceiverTagValue)

	return consumeTraceData(ctxWithReceiverName, batches, jr.nextConsumer)
}

func (jtr *jTchannelReceiver) SubmitBatches(ctx thrift.Context, batches []*jaeger.Batch) ([]*jaeger.BatchSubmitResponse, error) {
	ctxWithReceiverName := observability.ContextWithReceiverName(ctx, tchannelCollectorReceiverTagValue)

	return consumeTraceData(ctxWithReceiverName, batches, jtr.nextConsumer)
}

var _ reporter.Reporter = (*jReceiver)(nil)
var _ agentapp.CollectorProxy = (*jReceiver)(nil)
var _ api_v2.CollectorServiceServer = (*jReceiver)(nil)

// EmitZipkinBatch implements cmd/agent/reporter.Reporter and it forwards
// Zipkin spans received by the Jaeger agent processor.
func (jr *jReceiver) EmitZipkinBatch(spans []*zipkincore.Span) error {
	return nil
}

// EmitBatch implements cmd/agent/reporter.Reporter and it forwards
// Jaeger spans received by the Jaeger agent processor.
func (jr *jReceiver) EmitBatch(batch *jaeger.Batch) error {
	td, err := jaegertranslator.ThriftBatchToOCProto(batch)
	if err != nil {
		observability.RecordMetricsForTraceReceiver(jr.defaultAgentCtx, len(batch.Spans), len(batch.Spans))
		return err
	}

	err = jr.nextConsumer.ConsumeTraceData(jr.defaultAgentCtx, td)
	observability.RecordMetricsForTraceReceiver(jr.defaultAgentCtx, len(batch.Spans), len(batch.Spans)-len(td.Spans))

	return err
}

func (jr *jReceiver) GetReporter() reporter.Reporter {
	return jr
}

func (jr *jReceiver) GetManager() configmanager.ClientConfigManager {
	return jr
}

func (jr *jReceiver) GetSamplingStrategy(serviceName string) (*sampling.SamplingStrategyResponse, error) {
	return &sampling.SamplingStrategyResponse{}, nil
}

func (jr *jReceiver) GetBaggageRestrictions(serviceName string) ([]*baggage.BaggageRestriction, error) {
	return nil, nil
}

func (jr *jReceiver) PostSpans(ctx context.Context, r *api_v2.PostSpansRequest) (*api_v2.PostSpansResponse, error) {
	ctxWithReceiverName := observability.ContextWithReceiverName(ctx, collectorReceiverTagValue)

	td, err := jaegertranslator.ProtoBatchToOCProto(r.Batch)
	td.SourceFormat = "jaeger"
	if err != nil {
		observability.RecordMetricsForTraceReceiver(ctxWithReceiverName, len(r.Batch.Spans), len(r.Batch.Spans))
		return nil, err
	}

	err = jr.nextConsumer.ConsumeTraceData(ctx, td)
	observability.RecordMetricsForTraceReceiver(ctxWithReceiverName, len(r.Batch.Spans), len(r.Batch.Spans)-len(td.Spans))
	if err != nil {
		return nil, err
	}

	return &api_v2.PostSpansResponse{}, err
}

func (jr *jReceiver) startAgent(_ receiver.Host) error {
	processorConfigs := []agentapp.ProcessorConfiguration{
		{
			// Compact Thrift running by default on 6831.
			Model:    "jaeger",
			Protocol: "compact",
			Server: agentapp.ServerConfiguration{
				HostPort: jr.agentCompactThriftAddr(),
			},
		},
		{
			// Binary Thrift running by default on 6832.
			Model:    "jaeger",
			Protocol: "binary",
			Server: agentapp.ServerConfiguration{
				HostPort: jr.agentBinaryThriftAddr(),
			},
		},
	}

	builder := agentapp.Builder{
		Processors: processorConfigs,
		HTTPServer: agentapp.HTTPServerConfiguration{
			HostPort: jr.agentAddress(),
		},
	}

	agent, err := builder.CreateAgent(jr, zap.NewNop(), metrics.NullFactory)
	if err != nil {
		return err
	}

	if err := agent.Run(); err != nil {
		return err
	}

	// Otherwise no error was encountered,
	jr.agent = agent

	return nil
}

func (jr *jReceiver) startCollector(host receiver.Host) error {
	tch, terr := tchannel.NewChannel("jaeger-collector", new(tchannel.ChannelOptions))
	if terr != nil {
		return fmt.Errorf("failed to create NewTChannel: %v", terr)
	}

	server := thrift.NewServer(tch)
	server.Register(jaeger.NewTChanCollectorServer(jr.tchanServer))

	taddr := jr.tchannelAddr()
	tln, terr := net.Listen("tcp", taddr)
	if terr != nil {
		return fmt.Errorf("failed to bind to TChannel address %q: %v", taddr, terr)
	}
	tch.Serve(tln)
	jr.tchanServer.tchannel = tch

	// Now the collector that runs over HTTP
	caddr := jr.collectorAddr()
	cln, cerr := net.Listen("tcp", caddr)
	if cerr != nil {
		// Abort and close tch
		tch.Close()
		return fmt.Errorf("failed to bind to Collector address %q: %v", caddr, cerr)
	}

	nr := mux.NewRouter()
	apiHandler := app.NewAPIHandler(jr)
	apiHandler.RegisterRoutes(nr)
	jr.collectorServer = &http.Server{Handler: nr}
	go func() {
		_ = jr.collectorServer.Serve(cln)
	}()

	jr.grpc = grpc.NewServer(jr.config.CollectorGRPCOptions...)
	gaddr := jr.grpcAddr()
	gln, gerr := net.Listen("tcp", gaddr)
	if gerr != nil {
		// Abort and close tch, cln
		tch.Close()
		cln.Close()
		return fmt.Errorf("failed to bind to gRPC address %q: %v", gaddr, gerr)
	}

	api_v2.RegisterCollectorServiceServer(jr.grpc, jr)

	go func() {
		if err := jr.grpc.Serve(gln); err != nil {
			host.ReportFatalError(err)
		}
	}()

	return nil
}
