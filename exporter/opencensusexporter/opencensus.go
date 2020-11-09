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

package opencensusexporter

import (
	"context"
	"errors"
	"fmt"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	agentmetricspb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/metrics/v1"
	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/translator/internaldata"
)

// See https://godoc.org/google.golang.org/grpc#ClientConn.NewStream
// why we need to keep the cancel func to cancel the stream
type tracesClientWithCancel struct {
	cancel context.CancelFunc
	tsec   agenttracepb.TraceService_ExportClient
}

// See https://godoc.org/google.golang.org/grpc#ClientConn.NewStream
// why we need to keep the cancel func to cancel the stream
type metricsClientWithCancel struct {
	cancel context.CancelFunc
	msec   agentmetricspb.MetricsService_ExportClient
}

type ocExporter struct {
	cfg *Config
	// gRPC clients and connection.
	traceSvcClient   agenttracepb.TraceServiceClient
	metricsSvcClient agentmetricspb.MetricsServiceClient
	// In any of the channels we keep always NumWorkers object (sometimes nil),
	// to make sure we don't open more than NumWorkers RPCs at any moment.
	tracesClients  chan *tracesClientWithCancel
	metricsClients chan *metricsClientWithCancel
	grpcClientConn *grpc.ClientConn
	metadata       metadata.MD
}

func newOcExporter(ctx context.Context, cfg *Config) (*ocExporter, error) {
	if cfg.Endpoint == "" {
		return nil, errors.New("OpenCensus exporter cfg requires an Endpoint")
	}

	if cfg.NumWorkers <= 0 {
		return nil, errors.New("OpenCensus exporter cfg requires at least one worker")
	}

	dialOpts, err := cfg.GRPCClientSettings.ToDialOptions()
	if err != nil {
		return nil, err
	}

	var clientConn *grpc.ClientConn
	if clientConn, err = grpc.DialContext(ctx, cfg.GRPCClientSettings.Endpoint, dialOpts...); err != nil {
		return nil, err
	}

	oce := &ocExporter{
		cfg:            cfg,
		grpcClientConn: clientConn,
		metadata:       metadata.New(cfg.GRPCClientSettings.Headers),
	}
	return oce, nil
}

func (oce *ocExporter) shutdown(context.Context) error {
	if oce.tracesClients != nil {
		// First remove all the clients from the channel.
		for i := 0; i < oce.cfg.NumWorkers; i++ {
			<-oce.tracesClients
		}
		// Now close the channel
		close(oce.tracesClients)
	}
	if oce.metricsClients != nil {
		// First remove all the clients from the channel.
		for i := 0; i < oce.cfg.NumWorkers; i++ {
			<-oce.metricsClients
		}
		// Now close the channel
		close(oce.metricsClients)
	}
	return oce.grpcClientConn.Close()
}

func newTraceExporter(ctx context.Context, cfg *Config, logger *zap.Logger) (component.TracesExporter, error) {
	oce, err := newOcExporter(ctx, cfg)
	if err != nil {
		return nil, err
	}
	oce.traceSvcClient = agenttracepb.NewTraceServiceClient(oce.grpcClientConn)
	oce.tracesClients = make(chan *tracesClientWithCancel, cfg.NumWorkers)
	// Try to create rpc clients now.
	for i := 0; i < cfg.NumWorkers; i++ {
		// Populate the channel with NumWorkers nil RPCs to keep the number of workers
		// constant in the channel.
		oce.tracesClients <- nil
	}

	return exporterhelper.NewTraceExporter(
		cfg,
		logger,
		oce.pushTraceData,
		exporterhelper.WithShutdown(oce.shutdown))
}

func newMetricsExporter(ctx context.Context, cfg *Config, logger *zap.Logger) (component.MetricsExporter, error) {
	oce, err := newOcExporter(ctx, cfg)
	if err != nil {
		return nil, err
	}
	oce.metricsSvcClient = agentmetricspb.NewMetricsServiceClient(oce.grpcClientConn)
	oce.metricsClients = make(chan *metricsClientWithCancel, cfg.NumWorkers)
	// Try to create rpc clients now.
	for i := 0; i < cfg.NumWorkers; i++ {
		// Populate the channel with NumWorkers nil RPCs to keep the number of workers
		// constant in the channel.
		oce.metricsClients <- nil
	}

	return exporterhelper.NewMetricsExporter(
		cfg,
		logger,
		oce.pushMetricsData,
		exporterhelper.WithShutdown(oce.shutdown))
}

func (oce *ocExporter) pushTraceData(_ context.Context, td pdata.Traces) (int, error) {
	// Get first available trace Client.
	tClient, ok := <-oce.tracesClients
	if !ok {
		err := errors.New("failed to push traces, OpenCensus exporter was already stopped")
		return td.SpanCount(), err
	}

	// In any of the metricsClients channel we keep always NumWorkers object (sometimes nil),
	// to make sure we don't open more than NumWorkers RPCs at any moment.
	// Here check if the client is nil and create a new one if that is the case. A nil
	// object means that an error happened: could not connect, service went down, etc.
	if tClient == nil {
		var err error
		tClient, err = oce.createTraceServiceRPC()
		if err != nil {
			// Cannot create an RPC, put back nil to keep the number of workers constant.
			oce.tracesClients <- nil
			return td.SpanCount(), err
		}
	}

	octds := internaldata.TraceDataToOC(td)
	for _, octd := range octds {
		// This is a hack because OC protocol expects a Node for the initial message.
		node := octd.Node
		if node == nil {
			node = &commonpb.Node{}
		}
		resource := octd.Resource
		if resource == nil {
			resource = &resourcepb.Resource{}
		}
		req := &agenttracepb.ExportTraceServiceRequest{
			Spans:    octd.Spans,
			Resource: resource,
			Node:     node,
		}
		if err := tClient.tsec.Send(req); err != nil {
			// Error received, cancel the context used to create the RPC to free all resources,
			// put back nil to keep the number of workers constant.
			tClient.cancel()
			oce.tracesClients <- nil
			return td.SpanCount(), err
		}
	}
	oce.tracesClients <- tClient
	return 0, nil
}

func (oce *ocExporter) pushMetricsData(_ context.Context, md pdata.Metrics) (int, error) {
	// Get first available mClient.
	mClient, ok := <-oce.metricsClients
	if !ok {
		err := errors.New("failed to push metrics, OpenCensus exporter was already stopped")
		return metricPointCount(md), err
	}

	// In any of the metricsClients channel we keep always NumWorkers object (sometimes nil),
	// to make sure we don't open more than NumWorkers RPCs at any moment.
	// Here check if the client is nil and create a new one if that is the case. A nil
	// object means that an error happened: could not connect, service went down, etc.
	if mClient == nil {
		var err error
		mClient, err = oce.createMetricsServiceRPC()
		if err != nil {
			// Cannot create an RPC, put back nil to keep the number of workers constant.
			oce.metricsClients <- nil
			return metricPointCount(md), err
		}
	}

	ocmds := internaldata.MetricsToOC(md)
	for _, ocmd := range ocmds {
		// This is a hack because OC protocol expects a Node for the initial message.
		node := ocmd.Node
		if node == nil {
			node = &commonpb.Node{}
		}
		resource := ocmd.Resource
		if resource == nil {
			resource = &resourcepb.Resource{}
		}
		req := &agentmetricspb.ExportMetricsServiceRequest{
			Metrics:  ocmd.Metrics,
			Resource: resource,
			Node:     node,
		}
		if err := mClient.msec.Send(req); err != nil {
			// Error received, cancel the context used to create the RPC to free all resources,
			// put back nil to keep the number of workers constant.
			mClient.cancel()
			oce.metricsClients <- nil
			return metricPointCount(md), err
		}
	}
	oce.metricsClients <- mClient
	return 0, nil
}

func (oce *ocExporter) createTraceServiceRPC() (*tracesClientWithCancel, error) {
	// Initiate the trace service by sending over node identifier info.
	ctx, cancel := context.WithCancel(context.Background())
	if len(oce.cfg.Headers) > 0 {
		ctx = metadata.NewOutgoingContext(ctx, metadata.New(oce.cfg.Headers))
	}
	// Cannot use grpc.WaitForReady(cfg.WaitForReady) because will block forever.
	traceClient, err := oce.traceSvcClient.Export(ctx)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("TraceServiceClient: %w", err)
	}
	return &tracesClientWithCancel{cancel: cancel, tsec: traceClient}, nil
}

func (oce *ocExporter) createMetricsServiceRPC() (*metricsClientWithCancel, error) {
	// Initiate the trace service by sending over node identifier info.
	ctx, cancel := context.WithCancel(context.Background())
	if len(oce.cfg.Headers) > 0 {
		ctx = metadata.NewOutgoingContext(ctx, metadata.New(oce.cfg.Headers))
	}
	// Cannot use grpc.WaitForReady(cfg.WaitForReady) because will block forever.
	metricsClient, err := oce.metricsSvcClient.Export(ctx)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("MetricsServiceClient: %w", err)
	}
	return &metricsClientWithCancel{cancel: cancel, msec: metricsClient}, nil
}

func metricPointCount(md pdata.Metrics) int {
	_, pc := md.MetricAndDataPointCount()
	return pc
}
