// Copyright 2018, OpenCensus Authors
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

package opencensus

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"google.golang.org/grpc"

	"github.com/census-instrumentation/opencensus-service/internal"
	"github.com/census-instrumentation/opencensus-service/metricsink"
	"github.com/census-instrumentation/opencensus-service/receiver"
	"github.com/census-instrumentation/opencensus-service/receiver/opencensus/ocmetrics"
	"github.com/census-instrumentation/opencensus-service/receiver/opencensus/octrace"
	"github.com/census-instrumentation/opencensus-service/spansink"

	agentmetricspb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/metrics/v1"
	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
)

type ocReceiver struct {
	mu     sync.Mutex
	ln     net.Listener
	server *grpc.Server

	traceReceiver   *octrace.Receiver
	metricsReceiver *ocmetrics.Receiver

	stopOnce                 sync.Once
	startServerOnce          sync.Once
	startTraceReceiverOnce   sync.Once
	startMetricsReceiverOnce sync.Once
}

// TraceMetricsReceiverStopper is an interface that implements:
// * receiver.TraceReceiver
// * receiver.MetricsReceiver
// * a Stop() error method
type TraceMetricsReceiverStopper interface {
	receiver.TraceReceiver
	receiver.MetricsReceiver
	Stop() error
}

var _ TraceMetricsReceiverStopper = (*ocReceiver)(nil)

var (
	errAlreadyStarted = errors.New("already started")
	errAlreadyStopped = errors.New("already stopped")
)

const defaultOCReceiverPort = 55678

// New just creates the OpenCensus receiver services. It is the caller's
// responsibility to invoke the respective Start*Reception methods as well
// as the various Stop*Reception methods or simply Stop to end it.
func New(addr string) (TraceMetricsReceiverStopper, error) {
	// TODO: (@odeke-em) use options to enable address binding changes.
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("Failed to bind to address %q: error: %v", addr, err)
	}
	ocr := &ocReceiver{ln: ln}

	return ocr, nil
}

func (ocr *ocReceiver) StartTraceReception(ctx context.Context, ts spansink.Sink) error {
	var err = errAlreadyStarted

	ocr.startTraceReceiverOnce.Do(func() {
		ocr.traceReceiver, err = octrace.New(ts)
		if err == nil {
			srv := ocr.grpcServer()
			agenttracepb.RegisterTraceServiceServer(srv, ocr.traceReceiver)
		}
	})
	return err
}

func (ocr *ocReceiver) StartMetricsReception(ctx context.Context, ms metricsink.Sink) error {
	var err = errAlreadyStarted

	ocr.startMetricsReceiverOnce.Do(func() {
		ocr.metricsReceiver, err = ocmetrics.New(ms)
		if err == nil {
			srv := ocr.grpcServer()
			agentmetricspb.RegisterMetricsServiceServer(srv, ocr.metricsReceiver)
		}
	})
	return err
}

func (ocr *ocReceiver) grpcServer() *grpc.Server {
	ocr.mu.Lock()
	defer ocr.mu.Unlock()

	ocr.startServerOnce.Do(func() {
		ocr.server = internal.GRPCServerWithObservabilityEnabled()
		go func() {
			_ = ocr.server.Serve(ocr.ln)
		}()
	})
	return ocr.server
}

func (ocr *ocReceiver) StopTraceReception(ctx context.Context) error {
	// StopTraceReception is a noop currently.
	// TODO: (@odeke-em) investigate whether or not gRPC
	// provides a way to stop specific services.
	return nil
}

func (ocr *ocReceiver) StopMetricsReception(ctx context.Context) error {
	// StopMetricsReception is a noop currently.
	// TODO: (@odeke-em) investigate whether or not gRPC
	// provides a way to stop specific services.
	return nil
}

// TraceMetricsSink implements both spansink.Sink and metricsink.Sink.
type TraceMetricsSink interface {
	spansink.Sink
	metricsink.Sink
}

// Start runs all the receivers/services on it, invoking:
// * StartTraceReception
// * StartMetricsReception
func (ocr *ocReceiver) Start(ctx context.Context, sink TraceMetricsSink) error {
	if err := ocr.StartTraceReception(ctx, sink); err != nil && err != errAlreadyStarted {
		return err
	}
	if err := ocr.StartMetricsReception(ctx, sink); err != nil && err != errAlreadyStarted {
		return err
	}

	// At this point we've successfully started all the services/receivers.
	// Add other start routines here.
	return nil
}

// Stop stops the underlying gRPC server and all the services running on it.
func (ocr *ocReceiver) Stop() error {
	ocr.mu.Lock()
	defer ocr.mu.Unlock()

	var err = errAlreadyStopped
	ocr.stopOnce.Do(func() {
		// TODO: (@odeke-em) should we instead do (*grpc.Server).GracefulStop?
		ocr.server.Stop()
		_ = ocr.ln.Close()
	})
	return err
}
