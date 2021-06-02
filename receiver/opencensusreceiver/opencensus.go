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

package opencensusreceiver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"

	agentmetricspb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/metrics/v1"
	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
	gatewayruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/rs/cors"
	"github.com/soheilhy/cmux"
	"google.golang.org/grpc"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver/opencensusreceiver/internal/ocmetrics"
	"go.opentelemetry.io/collector/receiver/opencensusreceiver/internal/octrace"
)

// ocReceiver is the type that exposes Trace and Metrics reception.
type ocReceiver struct {
	mu                 sync.Mutex
	ln                 net.Listener
	serverGRPC         *grpc.Server
	serverHTTP         *http.Server
	gatewayMux         *gatewayruntime.ServeMux
	corsOrigins        []string
	grpcServerSettings configgrpc.GRPCServerSettings

	traceReceiver   *octrace.Receiver
	metricsReceiver *ocmetrics.Receiver

	traceConsumer   consumer.Traces
	metricsConsumer consumer.Metrics

	startTracesReceiverOnce  sync.Once
	startMetricsReceiverOnce sync.Once

	id config.ComponentID
}

// newOpenCensusReceiver just creates the OpenCensus receiver services. It is the caller's
// responsibility to invoke the respective Start*Reception methods as well
// as the various Stop*Reception methods to end it.
func newOpenCensusReceiver(
	id config.ComponentID,
	transport string,
	addr string,
	tc consumer.Traces,
	mc consumer.Metrics,
	opts ...ocOption,
) (*ocReceiver, error) {
	// TODO: (@odeke-em) use options to enable address binding changes.
	ln, err := net.Listen(transport, addr)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to address %q: %v", addr, err)
	}

	ocr := &ocReceiver{
		id:              id,
		ln:              ln,
		corsOrigins:     []string{}, // Disable CORS by default.
		gatewayMux:      gatewayruntime.NewServeMux(),
		traceConsumer:   tc,
		metricsConsumer: mc,
	}

	for _, opt := range opts {
		opt.withReceiver(ocr)
	}

	return ocr, nil
}

// Start runs the trace receiver on the gRPC server. Currently
// it also enables the metrics receiver too.
func (ocr *ocReceiver) Start(_ context.Context, host component.Host) error {
	hasConsumer := false
	if ocr.traceConsumer != nil {
		hasConsumer = true
		if err := ocr.registerTraceConsumer(host); err != nil {
			return err
		}
	}

	if ocr.metricsConsumer != nil {
		hasConsumer = true
		if err := ocr.registerMetricsConsumer(host); err != nil {
			return err
		}
	}

	if !hasConsumer {
		return errors.New("cannot start receiver: no consumers were specified")
	}

	if err := ocr.startServer(host); err != nil {
		return err
	}

	// At this point we've successfully started all the services/receivers.
	// Add other start routines here.
	return nil
}

func (ocr *ocReceiver) registerTraceConsumer(host component.Host) error {
	var err error

	ocr.startTracesReceiverOnce.Do(func() {
		ocr.traceReceiver, err = octrace.New(ocr.id, ocr.traceConsumer)
		if err != nil {
			return
		}

		var srv *grpc.Server
		srv, err = ocr.grpcServer(host)
		if err != nil {
			return
		}

		agenttracepb.RegisterTraceServiceServer(srv, ocr.traceReceiver)

	})

	return err
}

func (ocr *ocReceiver) registerMetricsConsumer(host component.Host) error {
	var err error

	ocr.startMetricsReceiverOnce.Do(func() {
		ocr.metricsReceiver, err = ocmetrics.New(ocr.id, ocr.metricsConsumer)
		if err != nil {
			return
		}

		var srv *grpc.Server
		srv, err = ocr.grpcServer(host)
		if err != nil {
			return
		}

		agentmetricspb.RegisterMetricsServiceServer(srv, ocr.metricsReceiver)
	})
	return err
}

func (ocr *ocReceiver) grpcServer(host component.Host) (*grpc.Server, error) {
	ocr.mu.Lock()
	defer ocr.mu.Unlock()

	if ocr.serverGRPC == nil {
		opts, err := ocr.grpcServerSettings.ToServerOption(host.GetExtensions())
		if err != nil {
			return nil, err
		}
		ocr.serverGRPC = grpc.NewServer(opts...)
	}

	return ocr.serverGRPC, nil
}

// Shutdown is a method to turn off receiving.
func (ocr *ocReceiver) Shutdown(context.Context) error {
	ocr.mu.Lock()
	defer ocr.mu.Unlock()

	var err error
	if ocr.serverHTTP != nil {
		err = ocr.serverHTTP.Close()
	}

	if ocr.ln != nil {
		_ = ocr.ln.Close()
	}

	// TODO: @(odeke-em) investigate what utility invoking (*grpc.Server).Stop()
	// gives us yet we invoke (net.Listener).Close().
	// Sure (*grpc.Server).Stop() enables proper shutdown but imposes
	// a painful and artificial wait time that goes into 20+seconds yet most of our
	// tests and code should be reactive in less than even 1second.
	// ocr.serverGRPC.Stop()

	return err
}

func (ocr *ocReceiver) httpServer() *http.Server {
	ocr.mu.Lock()
	defer ocr.mu.Unlock()

	if ocr.serverHTTP == nil {
		var mux http.Handler = ocr.gatewayMux
		if len(ocr.corsOrigins) > 0 {
			co := cors.Options{AllowedOrigins: ocr.corsOrigins}
			mux = cors.New(co).Handler(mux)
		}
		ocr.serverHTTP = &http.Server{Handler: mux}
	}

	return ocr.serverHTTP
}

func (ocr *ocReceiver) startServer(host component.Host) error {
	// Register the grpc-gateway on the HTTP server mux
	c := context.Background()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	endpoint := ocr.ln.Addr().String()

	_, ok := ocr.ln.(*net.UnixListener)
	if ok {
		endpoint = "unix:" + endpoint
	}

	if err := agenttracepb.RegisterTraceServiceHandlerFromEndpoint(c, ocr.gatewayMux, endpoint, opts); err != nil {
		return err
	}

	if err := agentmetricspb.RegisterMetricsServiceHandlerFromEndpoint(c, ocr.gatewayMux, endpoint, opts); err != nil {
		return err
	}

	// Start the gRPC and HTTP/JSON (grpc-gateway) servers on the same port.
	m := cmux.New(ocr.ln)
	grpcL := m.MatchWithWriters(
		cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"),
		cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc+proto"))

	httpL := m.Match(cmux.Any())
	go func() {
		if errGrpc := ocr.serverGRPC.Serve(grpcL); errGrpc != nil {
			host.ReportFatalError(errGrpc)
		}
	}()
	go func() {
		if errHTTP := ocr.httpServer().Serve(httpL); errHTTP != http.ErrServerClosed {
			host.ReportFatalError(errHTTP)
		}
	}()
	go func() {
		if errServe := m.Serve(); errServe != nil {
			host.ReportFatalError(errServe)
		}
	}()
	return nil
}
