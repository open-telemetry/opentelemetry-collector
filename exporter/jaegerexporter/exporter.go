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
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/model/pdata"
	jaegertranslator "go.opentelemetry.io/collector/translator/trace/jaeger"
)

// newTracesExporter returns a new Jaeger gRPC exporter.
// The exporter name is the name to be used in the observability of the exporter.
// The collectorEndpoint should be of the form "hostname:14250" (a gRPC target).
func newTracesExporter(cfg *Config, set component.ExporterCreateSettings) (component.TracesExporter, error) {
	s := newProtoGRPCSender(cfg, set.Logger)
	return exporterhelper.NewTracesExporter(
		cfg, set, s.pushTraces,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithStart(s.start),
		exporterhelper.WithShutdown(s.shutdown),
		exporterhelper.WithTimeout(cfg.TimeoutSettings),
		exporterhelper.WithRetry(cfg.RetrySettings),
		exporterhelper.WithQueue(cfg.QueueSettings),
	)
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

	stopCh         chan struct{}
	stopped        bool
	stopLock       sync.Mutex
	clientSettings *configgrpc.GRPCClientSettings
}

func newProtoGRPCSender(cfg *Config, logger *zap.Logger) *protoGRPCSender {
	s := &protoGRPCSender{
		name:                      cfg.ID().String(),
		logger:                    logger,
		metadata:                  metadata.New(cfg.GRPCClientSettings.Headers),
		waitForReady:              cfg.WaitForReady,
		connStateReporterInterval: time.Second,
		stopCh:                    make(chan struct{}),
		clientSettings:            &cfg.GRPCClientSettings,
	}
	s.AddStateChangeCallback(s.onStateChange)
	return s
}

type stateReporter interface {
	GetState() connectivity.State
}

func (s *protoGRPCSender) pushTraces(
	ctx context.Context,
	td pdata.Traces,
) error {

	batches, err := jaegertranslator.InternalTracesToJaegerProto(td)
	if err != nil {
		return consumererror.Permanent(fmt.Errorf("failed to push trace data via Jaeger exporter: %w", err))
	}

	if s.metadata.Len() > 0 {
		ctx = metadata.NewOutgoingContext(ctx, s.metadata)
	}

	for _, batch := range batches {
		_, err = s.client.PostSpans(
			ctx,
			&jaegerproto.PostSpansRequest{Batch: *batch}, grpc.WaitForReady(s.waitForReady))

		if err != nil {
			s.logger.Debug("failed to push trace data to Jaeger", zap.Error(err))
			return fmt.Errorf("failed to push trace data via Jaeger exporter: %w", err)
		}
	}

	return nil
}

func (s *protoGRPCSender) shutdown(context.Context) error {
	s.stopLock.Lock()
	s.stopped = true
	s.stopLock.Unlock()
	close(s.stopCh)
	return nil
}

func (s *protoGRPCSender) start(_ context.Context, host component.Host) error {
	if s.clientSettings == nil {
		return fmt.Errorf("client settings not found")
	}
	opts, err := s.clientSettings.ToDialOptions(host.GetExtensions())
	if err != nil {
		return err
	}

	conn, err := grpc.Dial(s.clientSettings.Endpoint, opts...)
	if err != nil {
		return err
	}

	s.client = jaegerproto.NewCollectorServiceClient(conn)
	s.conn = conn

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
