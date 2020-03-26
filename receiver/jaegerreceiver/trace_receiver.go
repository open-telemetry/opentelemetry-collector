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

	apacheThrift "github.com/apache/thrift/lib/go/thrift"
	"github.com/gorilla/mux"
	"github.com/jaegertracing/jaeger/cmd/agent/app/configmanager"
	jSamplingConfig "github.com/jaegertracing/jaeger/cmd/agent/app/configmanager/grpc"
	"github.com/jaegertracing/jaeger/cmd/agent/app/httpserver"
	"github.com/jaegertracing/jaeger/cmd/agent/app/processors"
	"github.com/jaegertracing/jaeger/cmd/agent/app/reporter"
	"github.com/jaegertracing/jaeger/cmd/agent/app/servers"
	"github.com/jaegertracing/jaeger/cmd/agent/app/servers/thriftudp"
	"github.com/jaegertracing/jaeger/cmd/collector/app/handler"
	collectorSampling "github.com/jaegertracing/jaeger/cmd/collector/app/sampling"
	staticStrategyStore "github.com/jaegertracing/jaeger/plugin/sampling/strategystore/static"
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

	"github.com/open-telemetry/opentelemetry-collector/client"
	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/obsreport"
	"github.com/open-telemetry/opentelemetry-collector/oterr"
	jaegertranslator "github.com/open-telemetry/opentelemetry-collector/translator/trace/jaeger"
)

var (
	batchSubmitNotOkResponse = &jaeger.BatchSubmitResponse{}
	batchSubmitOkResponse    = &jaeger.BatchSubmitResponse{Ok: true}
)

// Configuration defines the behavior and the ports that
// the Jaeger receiver will use.
type Configuration struct {
	CollectorThriftPort  int
	CollectorHTTPPort    int
	CollectorGRPCPort    int
	CollectorGRPCOptions []grpc.ServerOption

	AgentCompactThriftPort     int
	AgentBinaryThriftPort      int
	AgentHTTPPort              int
	RemoteSamplingEndpoint     string
	RemoteSamplingStrategyFile string
}

// Receiver type is used to receive spans that were originally intended to be sent to Jaeger.
// This receiver is basically a Jaeger collector.
type jReceiver struct {
	// mu protects the fields of this type
	mu sync.Mutex

	nextConsumer consumer.TraceConsumerOld
	instanceName string

	startOnce sync.Once
	stopOnce  sync.Once

	config *Configuration

	grpc            *grpc.Server
	tchanServer     *jTchannelReceiver
	collectorServer *http.Server

	agentSamplingManager *jSamplingConfig.SamplingManager
	agentProcessors      []processors.Processor
	agentServer          *http.Server

	defaultAgentCtx context.Context
	logger          *zap.Logger
}

type jTchannelReceiver struct {
	nextConsumer consumer.TraceConsumerOld
	instanceName string

	tchannel *tchannel.Channel
}

const (
	defaultAgentQueueSize     = 1000
	defaultAgentMaxPacketSize = 65000
	defaultAgentServerWorkers = 10

	// Legacy metrics receiver name tag values
	collectorReceiverTagValue         = "jaeger-collector"
	tchannelCollectorReceiverTagValue = "jaeger-tchannel-collector"
	agentReceiverTagValue             = "jaeger-agent"

	agentTransport             = "agent" // This is not 100% precise since it can be either compact or binary.
	collectorHTTPTransport     = "collector_http"
	collectorTChannelTransport = "collector_tchannel"
	grpcTransport              = "grpc"

	thriftFormat   = "thrift"
	protobufFormat = "protobuf"
)

// New creates a TraceReceiver that receives traffic as a Jaeger collector, and
// also as a Jaeger agent.
func New(
	instanceName string,
	config *Configuration,
	nextConsumer consumer.TraceConsumerOld,
	logger *zap.Logger,
) (component.TraceReceiver, error) {
	return &jReceiver{
		config: config,
		defaultAgentCtx: obsreport.ReceiverContext(
			context.Background(), instanceName, agentTransport, agentReceiverTagValue),
		nextConsumer: nextConsumer,
		instanceName: instanceName,
		tchanServer: &jTchannelReceiver{
			nextConsumer: nextConsumer,
			instanceName: instanceName,
		},
		logger: logger,
	}, nil
}

func (jr *jReceiver) agentCompactThriftAddr() string {
	var port int
	if jr.config != nil {
		port = jr.config.AgentCompactThriftPort
	}
	return fmt.Sprintf(":%d", port)
}

func (jr *jReceiver) agentCompactThriftEnabled() bool {
	return jr.config != nil && jr.config.AgentCompactThriftPort > 0
}

func (jr *jReceiver) agentBinaryThriftAddr() string {
	var port int
	if jr.config != nil {
		port = jr.config.AgentBinaryThriftPort
	}
	return fmt.Sprintf(":%d", port)
}

func (jr *jReceiver) agentBinaryThriftEnabled() bool {
	return jr.config != nil && jr.config.AgentBinaryThriftPort > 0
}

func (jr *jReceiver) agentHTTPAddr() string {
	var port int
	if jr.config != nil {
		port = jr.config.AgentHTTPPort
	}
	return fmt.Sprintf(":%d", port)
}

func (jr *jReceiver) agentHTTPEnabled() bool {
	return jr.config != nil && jr.config.AgentHTTPPort > 0
}

func (jr *jReceiver) collectorGRPCAddr() string {
	var port int
	if jr.config != nil {
		port = jr.config.CollectorGRPCPort
	}
	return fmt.Sprintf(":%d", port)
}

func (jr *jReceiver) collectorGRPCEnabled() bool {
	return jr.config != nil && jr.config.CollectorGRPCPort > 0
}

func (jr *jReceiver) collectorHTTPAddr() string {
	var port int
	if jr.config != nil {
		port = jr.config.CollectorHTTPPort
	}
	return fmt.Sprintf(":%d", port)
}

func (jr *jReceiver) collectorHTTPEnabled() bool {
	return jr.config != nil && jr.config.CollectorHTTPPort > 0
}

// TODO https://github.com/open-telemetry/opentelemetry-collector/issues/267
//	Remove ThriftTChannel support.
func (jr *jReceiver) collectorThriftAddr() string {
	var port int
	if jr.config != nil {
		port = jr.config.CollectorThriftPort
	}
	return fmt.Sprintf(":%d", port)
}

func (jr *jReceiver) collectorThriftEnabled() bool {
	return jr.config != nil && jr.config.CollectorThriftPort > 0
}

func (jr *jReceiver) Start(host component.Host) error {
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

func (jr *jReceiver) Shutdown() error {
	jr.mu.Lock()
	defer jr.mu.Unlock()

	return jr.stopTraceReceptionLocked()
}

func (jr *jReceiver) stopTraceReceptionLocked() error {
	var err = oterr.ErrAlreadyStopped
	jr.stopOnce.Do(func() {
		var errs []error

		if jr.agentServer != nil {
			if aerr := jr.agentServer.Close(); aerr != nil {
				errs = append(errs, aerr)
			}
			jr.agentServer = nil
		}
		for _, processor := range jr.agentProcessors {
			processor.Stop()
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

func consumeTraceData(
	ctx context.Context,
	batches []*jaeger.Batch,
	consumer consumer.TraceConsumerOld,
) ([]*jaeger.BatchSubmitResponse, int, error) {

	jbsr := make([]*jaeger.BatchSubmitResponse, 0, len(batches))
	var consumerError error
	numSpans := 0
	for _, batch := range batches {
		numSpans += len(batch.Spans)
		if consumerError != nil {
			jbsr = append(jbsr, batchSubmitNotOkResponse)
			continue
		}

		// TODO: function below never returns error, change the signature.
		td, _ := jaegertranslator.ThriftBatchToOCProto(batch)
		td.SourceFormat = "jaeger"
		consumerError = consumer.ConsumeTraceData(ctx, td)
		jsr := batchSubmitOkResponse
		if consumerError != nil {
			jsr = batchSubmitNotOkResponse
		}
		jbsr = append(jbsr, jsr)
	}

	return jbsr, numSpans, consumerError
}

func (jr *jReceiver) SubmitBatches(batches []*jaeger.Batch, options handler.SubmitBatchOptions) ([]*jaeger.BatchSubmitResponse, error) {
	ctx := obsreport.ReceiverContext(
		context.Background(), jr.instanceName, collectorHTTPTransport, collectorReceiverTagValue)
	ctx = obsreport.StartTraceDataReceiveOp(
		ctx, jr.instanceName, collectorHTTPTransport)

	jbsr, numSpans, err := consumeTraceData(ctx, batches, jr.nextConsumer)
	obsreport.EndTraceDataReceiveOp(ctx, thriftFormat, numSpans, err)

	return jbsr, err
}

func (jtr *jTchannelReceiver) SubmitBatches(thriftCtx thrift.Context, batches []*jaeger.Batch) ([]*jaeger.BatchSubmitResponse, error) {
	ctx := obsreport.ReceiverContext(
		thriftCtx, jtr.instanceName, collectorTChannelTransport, tchannelCollectorReceiverTagValue)
	ctx = obsreport.StartTraceDataReceiveOp(ctx, jtr.instanceName, collectorTChannelTransport)

	jbsr, numSpans, err := consumeTraceData(ctx, batches, jtr.nextConsumer)
	obsreport.EndTraceDataReceiveOp(ctx, thriftFormat, numSpans, err)

	return jbsr, err
}

var _ reporter.Reporter = (*jReceiver)(nil)
var _ api_v2.CollectorServiceServer = (*jReceiver)(nil)
var _ configmanager.ClientConfigManager = (*jReceiver)(nil)

// EmitZipkinBatch implements cmd/agent/reporter.Reporter and it forwards
// Zipkin spans received by the Jaeger agent processor.
func (jr *jReceiver) EmitZipkinBatch(spans []*zipkincore.Span) error {
	return nil
}

// EmitBatch implements cmd/agent/reporter.Reporter and it forwards
// Jaeger spans received by the Jaeger agent processor.
func (jr *jReceiver) EmitBatch(batch *jaeger.Batch) error {
	ctx := obsreport.StartTraceDataReceiveOp(
		jr.defaultAgentCtx, jr.instanceName, agentTransport)
	// TODO: call below never returns error it remove from the signature
	td, _ := jaegertranslator.ThriftBatchToOCProto(batch)
	td.SourceFormat = "jaeger"

	err := jr.nextConsumer.ConsumeTraceData(ctx, td)
	obsreport.EndTraceDataReceiveOp(ctx, thriftFormat, len(batch.Spans), err)

	return err
}

func (jr *jReceiver) GetSamplingStrategy(serviceName string) (*sampling.SamplingStrategyResponse, error) {
	return jr.agentSamplingManager.GetSamplingStrategy(serviceName)
}

func (jr *jReceiver) GetBaggageRestrictions(serviceName string) ([]*baggage.BaggageRestriction, error) {
	br, err := jr.agentSamplingManager.GetBaggageRestrictions(serviceName)
	if err != nil {
		// Baggage restrictions are not yet implemented - refer to - https://github.com/jaegertracing/jaeger/issues/373
		// As of today, GetBaggageRestrictions() always returns an error.
		// However, we `return nil, nil` here in order to serve a valid `200 OK` response.
		return nil, nil
	}
	return br, nil
}

func (jr *jReceiver) PostSpans(ctx context.Context, r *api_v2.PostSpansRequest) (*api_v2.PostSpansResponse, error) {
	if c, ok := client.FromGRPC(ctx); ok {
		ctx = client.NewContext(ctx, c)
	}

	ctx = obsreport.ReceiverContext(
		ctx, jr.instanceName, grpcTransport, collectorReceiverTagValue)
	ctx = obsreport.StartTraceDataReceiveOp(ctx, jr.instanceName, grpcTransport)

	// TODO: the function below never returns error, change its interface.
	td, _ := jaegertranslator.ProtoBatchToOCProto(r.GetBatch())
	td.SourceFormat = "jaeger"

	err := jr.nextConsumer.ConsumeTraceData(ctx, td)
	obsreport.EndTraceDataReceiveOp(ctx, protobufFormat, len(r.GetBatch().Spans), err)
	if err != nil {
		return nil, err
	}

	return &api_v2.PostSpansResponse{}, nil
}

func (jr *jReceiver) startAgent(_ component.Host) error {
	if !jr.agentBinaryThriftEnabled() && !jr.agentCompactThriftEnabled() && !jr.agentHTTPEnabled() {
		return nil
	}

	if jr.agentBinaryThriftEnabled() {
		processor, err := jr.buildProcessor(jr.agentBinaryThriftAddr(), apacheThrift.NewTBinaryProtocolFactoryDefault())
		if err != nil {
			return err
		}
		jr.agentProcessors = append(jr.agentProcessors, processor)
	}

	if jr.agentCompactThriftEnabled() {
		processor, err := jr.buildProcessor(jr.agentCompactThriftAddr(), apacheThrift.NewTCompactProtocolFactory())
		if err != nil {
			return err
		}
		jr.agentProcessors = append(jr.agentProcessors, processor)
	}

	for _, processor := range jr.agentProcessors {
		go processor.Serve()
	}

	// Start upstream grpc client before serving sampling endpoints over HTTP
	if jr.config.RemoteSamplingEndpoint != "" {
		conn, err := grpc.Dial(jr.config.RemoteSamplingEndpoint, grpc.WithInsecure())
		if err != nil {
			jr.logger.Error("Error creating grpc connection to jaeger remote sampling endpoint", zap.String("endpoint", jr.config.RemoteSamplingEndpoint))
			return err
		}

		jr.agentSamplingManager = jSamplingConfig.NewConfigManager(conn)
	}

	if jr.agentHTTPEnabled() {
		jr.agentServer = httpserver.NewHTTPServer(jr.agentHTTPAddr(), jr, metrics.NullFactory)

		go func() {
			if err := jr.agentServer.ListenAndServe(); err != nil {
				jr.logger.Error("http server failure", zap.Error(err))
			}
		}()
	}

	return nil
}

func (jr *jReceiver) buildProcessor(address string, factory apacheThrift.TProtocolFactory) (processors.Processor, error) {
	handler := jaeger.NewAgentProcessor(jr)
	transport, err := thriftudp.NewTUDPServerTransport(address)
	if err != nil {
		return nil, err
	}
	server, err := servers.NewTBufferedServer(transport, defaultAgentQueueSize, defaultAgentMaxPacketSize, metrics.NullFactory)
	if err != nil {
		return nil, err
	}
	processor, err := processors.NewThriftProcessor(server, defaultAgentServerWorkers, metrics.NullFactory, factory, handler, jr.logger)
	if err != nil {
		return nil, err
	}
	return processor, nil
}

func (jr *jReceiver) startCollector(host component.Host) error {
	if !jr.collectorGRPCEnabled() && !jr.collectorHTTPEnabled() && !jr.collectorThriftEnabled() {
		return nil
	}

	if jr.collectorThriftEnabled() {
		tch, terr := tchannel.NewChannel("jaeger-collector", new(tchannel.ChannelOptions))
		if terr != nil {
			return fmt.Errorf("failed to create NewTChannel: %v", terr)
		}

		server := thrift.NewServer(tch)
		server.Register(jaeger.NewTChanCollectorServer(jr.tchanServer))

		taddr := jr.collectorThriftAddr()
		tln, terr := net.Listen("tcp", taddr)
		if terr != nil {
			return fmt.Errorf("failed to bind to TChannel address %q: %v", taddr, terr)
		}
		tch.Serve(tln)
		jr.tchanServer.tchannel = tch
	}

	if jr.collectorHTTPEnabled() {
		// Now the collector that runs over HTTP
		caddr := jr.collectorHTTPAddr()
		cln, cerr := net.Listen("tcp", caddr)
		if cerr != nil {
			return fmt.Errorf("failed to bind to Collector address %q: %v", caddr, cerr)
		}

		nr := mux.NewRouter()
		apiHandler := handler.NewAPIHandler(jr)
		apiHandler.RegisterRoutes(nr)
		jr.collectorServer = &http.Server{Handler: nr}
		go func() {
			_ = jr.collectorServer.Serve(cln)
		}()
	}

	if jr.collectorGRPCEnabled() {
		jr.grpc = grpc.NewServer(jr.config.CollectorGRPCOptions...)
		gaddr := jr.collectorGRPCAddr()
		gln, gerr := net.Listen("tcp", gaddr)
		if gerr != nil {
			return fmt.Errorf("failed to bind to gRPC address %q: %v", gaddr, gerr)
		}

		api_v2.RegisterCollectorServiceServer(jr.grpc, jr)

		// init and register sampling strategy store
		if len(jr.config.RemoteSamplingStrategyFile) != 0 {
			ss, gerr := staticStrategyStore.NewStrategyStore(staticStrategyStore.Options{
				StrategiesFile: jr.config.RemoteSamplingStrategyFile,
			}, jr.logger)
			if gerr != nil {
				return fmt.Errorf("failed to create collector strategy store: %v", gerr)
			}
			api_v2.RegisterSamplingManagerServer(jr.grpc, collectorSampling.NewGRPCHandler(ss))
		}

		go func() {
			if err := jr.grpc.Serve(gln); err != nil {
				host.ReportFatalError(err)
			}
		}()
	}

	return nil
}
