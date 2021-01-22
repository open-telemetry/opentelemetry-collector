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

package jaegerexporter

import (
	"context"
	"fmt"
	"sync"
	"time"

	jaegerproto "github.com/jaegertracing/jaeger/proto-gen/api_v2"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/metadata"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	jaegertranslator "go.opentelemetry.io/collector/translator/trace/jaeger"
)

// newTraceExporter returns a new Jaeger gRPC exporter.
// The exporter name is the name to be used in the observability of the exporter.
// The collectorEndpoint should be of the form "hostname:14250" (a gRPC target).
func newTraceExporter(cfg *Config, logger *zap.Logger) (component.TracesExporter, error) {

	opts, err := cfg.GRPCClientSettings.ToDialOptions()
	if err != nil {
		return nil, err
	}

	conn, err := grpc.Dial(cfg.GRPCClientSettings.Endpoint, opts...)
	if err != nil {
		return nil, err
	}

	collectorServiceClient := jaegerproto.NewCollectorServiceClient(conn)
	s := newProtoGRPCSender(logger,
		cfg.NameVal,
		collectorServiceClient,
		metadata.New(cfg.GRPCClientSettings.Headers),
		cfg.WaitForReady,
		conn,
	)
	exp, err := exporterhelper.NewTraceExporter(
		cfg, logger, s.pushTraceData,
		exporterhelper.WithStart(s.start),
		exporterhelper.WithShutdown(s.shutdown),
		exporterhelper.WithTimeout(cfg.TimeoutSettings),
		exporterhelper.WithRetry(cfg.RetrySettings),
		exporterhelper.WithQueue(cfg.QueueSettings),
	)

	return exp, err
}

// protoGRPCSender forwards spans encoded in the jaeger proto
// format, to a grpc server.
type protoGRPCSender struct {
	name         string
	logger       *zap.Logger
	client       jaegerproto.CollectorServiceClient
	metadata     metadata.MD
	waitForReady bool

	conn                      stateReporter
	connStateReporterInterval time.Duration
	stateChangeCallbacks      []func(connectivity.State)

	stopCh   chan (struct{})
	stopped  bool
	stopLock sync.Mutex
}

func newProtoGRPCSender(logger *zap.Logger, name string, cl jaegerproto.CollectorServiceClient, md metadata.MD, waitForReady bool, conn stateReporter) *protoGRPCSender {
	s := &protoGRPCSender{
		name:         name,
		logger:       logger,
		client:       cl,
		metadata:     md,
		waitForReady: waitForReady,

		conn:                      conn,
		connStateReporterInterval: time.Second,

		stopCh: make(chan (struct{})),
	}
	s.AddStateChangeCallback(s.onStateChange)
	return s
}

type stateReporter interface {
	GetState() connectivity.State
}

func (s *protoGRPCSender) pushTraceData(
	ctx context.Context,
	td pdata.Traces,
) (droppedSpans int, err error) {

	batches, err := jaegertranslator.InternalTracesToJaegerProto(td)
	if err != nil {
		return td.SpanCount(), consumererror.Permanent(fmt.Errorf("failed to push trace data via Jaeger exporter: %w", err))
	}

	if s.metadata.Len() > 0 {
		ctx = metadata.NewOutgoingContext(ctx, s.metadata)
	}

	var sentSpans int
	for _, batch := range batches {
		_, err = s.client.PostSpans(
			ctx,
			&jaegerproto.PostSpansRequest{Batch: *batch}, grpc.WaitForReady(s.waitForReady))
		if err != nil {
			s.logger.Debug("failed to push trace data to Jaeger", zap.Error(err))
			return td.SpanCount() - sentSpans, fmt.Errorf("failed to push trace data via Jaeger exporter: %w", err)
		}
		sentSpans += len(batch.Spans)
	}

	return 0, nil
}

func (s *protoGRPCSender) shutdown(context.Context) error {
	s.stopLock.Lock()
	s.stopped = true
	s.stopLock.Unlock()
	close(s.stopCh)
	return nil
}

func (s *protoGRPCSender) start(context.Context, component.Host) error {
	go s.startConnectionStatusReporter()
	return nil
}

func (s *protoGRPCSender) startConnectionStatusReporter() {
	connState := s.conn.GetState()
	s.propagateStateChange(connState)

	ticker := time.NewTicker(s.connStateReporterInterval)
	for {
		select {
		case <-ticker.C:
			s.stopLock.Lock()
			if s.stopped {
				s.stopLock.Unlock()
				return
			}

			st := s.conn.GetState()
			if connState != st {
				// state has changed, report it
				connState = st
				s.propagateStateChange(st)
			}
			s.stopLock.Unlock()
		case <-s.stopCh:
			return
		}
	}
}

func (s *protoGRPCSender) propagateStateChange(st connectivity.State) {
	for _, callback := range s.stateChangeCallbacks {
		callback(st)
	}
}

func (s *protoGRPCSender) onStateChange(st connectivity.State) {
	mCtx, _ := tag.New(context.Background(), tag.Upsert(tag.MustNewKey("exporter_name"), s.name))
	stats.Record(mCtx, mLastConnectionState.M(int64(st)))
	s.logger.Info("State of the connection with the Jaeger Collector backend", zap.Stringer("state", st))
}

func (s *protoGRPCSender) AddStateChangeCallback(f func(connectivity.State)) {
	s.stateChangeCallbacks = append(s.stateChangeCallbacks, f)
}
