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

// Program occollector receives stats and traces from multiple sources and
// batches them for appropriate forwarding to backends (e.g.: Jaeger or Zipkin)
// or other layers of occollector. The forwarding can be configured so
// buffer sizes, number of retries, backoff policy, etc can be ajusted according
// to specific needs of each deployment.
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/census-instrumentation/opencensus-service/interceptor/octrace"
	"github.com/census-instrumentation/opencensus-service/spanreceiver"

	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats/view"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

const (
	defaultOCAddress = ":55678"
)

func main() {
	var signalsChannel = make(chan os.Signal)
	signal.Notify(signalsChannel, os.Interrupt, syscall.SIGTERM)

	// TODO: Allow configuration of logger and other items such as servers, etc.
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to get production logger: %v", err)
	}

	logger.Info("Starting...")

	closeSrv, err := runOCServerWithInterceptor(defaultOCAddress, logger)
	if err != nil {
		logger.Fatal("Cannot run opencensus server", zap.String("Address", defaultOCAddress), zap.Error(err))
	}

	logger.Info("Collector is up and running.")

	<-signalsChannel
	logger.Info("Starting shutdown...")

	// TODO: orderly shutdown: first receivers, then flushing pipelines giving
	// senders a chance to send all their data. This may take time, the allowed
	// time should be part of configuration.
	closeSrv()

	logger.Info("Shutdown complete.")
}

func runOCServerWithInterceptor(addr string, logger *zap.Logger) (func() error, error) {
	grpcSrv := grpc.NewServer()

	if err := view.Register(ocgrpc.DefaultServerViews...); err != nil {
		return nil, fmt.Errorf("Failed to register ocgrpc.DefaultServerViews: %v", err)
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("Cannot bind tcp listener to address %q: %v", addr, err)
	}

	sr := &fakeSpanReceiver{
		logger: logger,
	}

	oci, err := octrace.New(sr, octrace.WithSpanBufferPeriod(800*time.Millisecond))
	if err != nil {
		return nil, fmt.Errorf("Failed to create the OpenCensus interceptor: %v", err)
	}

	agenttracepb.RegisterTraceServiceServer(grpcSrv, oci)
	go func() {
		if err := grpcSrv.Serve(lis); err != nil {
			logger.Error("OpenCensus gRPC shutdown", zap.Error(err))
		}
	}()

	closeFn := func() error {
		grpcSrv.Stop()
		return nil
	}

	return closeFn, nil
}

type fakeSpanReceiver struct {
	logger *zap.Logger
}

func (sr *fakeSpanReceiver) ReceiveSpans(ctx context.Context, node *commonpb.Node, spans ...*tracepb.Span) (*spanreceiver.Acknowledgement, error) {
	ack := &spanreceiver.Acknowledgement{
		SavedSpans: uint64(len(spans)),
	}

	sr.logger.Info("ReceivedSpans", zap.Uint64("Received spans", ack.SavedSpans))

	return ack, nil
}
