// Copyright The OpenTelemetry Authors
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

package otlpreceiver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"

	gatewayruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/rs/cors"
	"github.com/soheilhy/cmux"
	"google.golang.org/grpc"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/consumer"
	collectormetrics "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/collector/metrics/v1"
	collectortrace "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/collector/trace/v1"
	"go.opentelemetry.io/collector/observability"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/metrics"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/trace"
)

// Receiver is the type that exposes Trace and Metrics reception.
type Receiver struct {
	mu                sync.Mutex
	ln                net.Listener
	serverGRPC        *grpc.Server
	serverHTTP        *http.Server
	gatewayMux        *gatewayruntime.ServeMux
	corsOrigins       []string
	grpcServerOptions []grpc.ServerOption

	traceReceiver   *trace.Receiver
	metricsReceiver *metrics.Receiver

	traceConsumer   consumer.TraceConsumer
	metricsConsumer consumer.MetricsConsumer

	stopOnce                 sync.Once
	startServerOnce          sync.Once
	startTraceReceiverOnce   sync.Once
	startMetricsReceiverOnce sync.Once

	instanceName string
}

// New just creates the OpenTelemetry receiver services. It is the caller's
// responsibility to invoke the respective Start*Reception methods as well
// as the various Stop*Reception methods to end it.
func New(
	instanceName string,
	transport string,
	addr string,
	tc consumer.TraceConsumer,
	mc consumer.MetricsConsumer,
	opts ...Option,
) (*Receiver, error) {
	ln, err := net.Listen(transport, addr)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to address %q: %v", addr, err)
	}

	r := &Receiver{
		ln:          ln,
		corsOrigins: []string{}, // Disable CORS by default.
		gatewayMux: gatewayruntime.NewServeMux(
			gatewayruntime.WithMarshalerOption("application/x-protobuf", &xProtobufMarshaler{}),
		),
	}

	for _, opt := range opts {
		opt.withReceiver(r)
	}

	r.instanceName = instanceName
	r.traceConsumer = tc
	r.metricsConsumer = mc

	return r, nil
}

// Start runs the trace receiver on the gRPC server. Currently
// it also enables the metrics receiver too.
func (r *Receiver) Start(ctx context.Context, host component.Host) error {
	return r.start(host)
}

func (r *Receiver) registerTraceConsumer() error {
	var err = componenterror.ErrAlreadyStarted

	r.startTraceReceiverOnce.Do(func() {
		r.traceReceiver, err = trace.New(r.instanceName, r.traceConsumer)
		if err != nil {
			return
		}
		srv := r.grpcServer()
		collectortrace.RegisterTraceServiceServer(srv, r.traceReceiver)
	})

	return err
}

func (r *Receiver) registerMetricsConsumer() error {
	var err = componenterror.ErrAlreadyStarted

	r.startMetricsReceiverOnce.Do(func() {
		r.metricsReceiver, err = metrics.New(r.instanceName, r.metricsConsumer)
		if err != nil {
			return
		}
		srv := r.grpcServer()
		collectormetrics.RegisterMetricsServiceServer(srv, r.metricsReceiver)
	})

	return err
}

func (r *Receiver) grpcServer() *grpc.Server {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.serverGRPC == nil {
		r.serverGRPC = observability.GRPCServerWithObservabilityEnabled(r.grpcServerOptions...)
	}

	return r.serverGRPC
}

// Shutdown is a method to turn off receiving.
func (r *Receiver) Shutdown(context.Context) error {
	if err := r.stop(); err != componenterror.ErrAlreadyStopped {
		return err
	}
	return nil
}

// start runs all the receivers/services namely, Trace and Metrics services.
func (r *Receiver) start(host component.Host) error {
	hasConsumer := false
	if r.traceConsumer != nil {
		hasConsumer = true
		if err := r.registerTraceConsumer(); err != nil && err != componenterror.ErrAlreadyStarted {
			return err
		}
	}

	if r.metricsConsumer != nil {
		hasConsumer = true
		if err := r.registerMetricsConsumer(); err != nil && err != componenterror.ErrAlreadyStarted {
			return err
		}
	}

	if !hasConsumer {
		return errors.New("cannot start receiver: no consumers were specified")
	}

	if err := r.startServer(host); err != nil && err != componenterror.ErrAlreadyStarted {
		return err
	}

	// At this point we've successfully started all the services/receivers.
	// Add other start routines here.
	return nil
}

// stop stops the underlying gRPC server and all the services running on it.
func (r *Receiver) stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var err = componenterror.ErrAlreadyStopped
	r.stopOnce.Do(func() {
		err = nil

		if r.serverHTTP != nil {
			_ = r.serverHTTP.Close()
		}

		if r.ln != nil {
			_ = r.ln.Close()
		}

		// TODO(nilebox): investigate, takes too long
		//  r.serverGRPC.Stop()
	})
	return err
}

func (r *Receiver) httpServer() *http.Server {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.serverHTTP == nil {
		var mux http.Handler = r.gatewayMux
		if len(r.corsOrigins) > 0 {
			co := cors.Options{AllowedOrigins: r.corsOrigins}
			mux = cors.New(co).Handler(mux)
		}
		r.serverHTTP = &http.Server{Handler: mux}
	}

	return r.serverHTTP
}

func (r *Receiver) startServer(host component.Host) error {
	err := componenterror.ErrAlreadyStarted
	r.startServerOnce.Do(func() {
		err = nil
		// Register the grpc-gateway on the HTTP server mux
		c := context.Background()
		opts := []grpc.DialOption{grpc.WithInsecure()}
		endpoint := r.ln.Addr().String()

		_, ok := r.ln.(*net.UnixListener)
		if ok {
			endpoint = "unix:" + endpoint
		}

		err = collectortrace.RegisterTraceServiceHandlerFromEndpoint(c, r.gatewayMux, endpoint, opts)
		if err != nil {
			return
		}

		err = collectormetrics.RegisterMetricsServiceHandlerFromEndpoint(c, r.gatewayMux, endpoint, opts)
		if err != nil {
			return
		}

		// Start the gRPC and HTTP/JSON (grpc-gateway) servers on the same port.
		m := cmux.New(r.ln)
		grpcL := m.MatchWithWriters(
			cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"),
			cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc+proto"))

		httpL := m.Match(cmux.Any())
		go func() {
			if errGrpc := r.serverGRPC.Serve(grpcL); errGrpc != nil {
				host.ReportFatalError(errGrpc)
			}
		}()
		go func() {
			if errHTTP := r.httpServer().Serve(httpL); errHTTP != nil {
				host.ReportFatalError(errHTTP)
			}
		}()
		go func() {
			if errServe := m.Serve(); errServe != nil {
				host.ReportFatalError(errServe)
			}
		}()
	})
	return err
}
