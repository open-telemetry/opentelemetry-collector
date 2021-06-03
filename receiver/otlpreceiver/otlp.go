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

package otlpreceiver

import (
	"context"
	"net"
	"net/http"
	"sync"

	gatewayruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/consumer"
	collectorlog "go.opentelemetry.io/collector/internal/data/protogen/collector/logs/v1"
	collectormetrics "go.opentelemetry.io/collector/internal/data/protogen/collector/metrics/v1"
	collectortrace "go.opentelemetry.io/collector/internal/data/protogen/collector/trace/v1"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/logs"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/metrics"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/trace"
)

// otlpReceiver is the type that exposes Trace and Metrics reception.
type otlpReceiver struct {
	cfg        *Config
	serverGRPC *grpc.Server
	gatewayMux *gatewayruntime.ServeMux
	serverHTTP *http.Server

	traceReceiver   *trace.Receiver
	metricsReceiver *metrics.Receiver
	logReceiver     *logs.Receiver
	shutdownWG      sync.WaitGroup

	logger *zap.Logger
}

// newOtlpReceiver just creates the OpenTelemetry receiver services. It is the caller's
// responsibility to invoke the respective Start*Reception methods as well
// as the various Stop*Reception methods to end it.
func newOtlpReceiver(cfg *Config, logger *zap.Logger) *otlpReceiver {
	r := &otlpReceiver{
		cfg:    cfg,
		logger: logger,
	}
	if cfg.HTTP != nil {
		// Use our custom JSON marshaler instead of default Protobuf JSON marshaler.
		// This is needed because OTLP spec defines encoding for trace and span id
		// and it is only possible to do using Gogoproto-compatible JSONPb marshaler.
		jsonpb := &internal.JSONPb{
			EmitDefaults: true,
			Indent:       "  ",
			OrigName:     true,
		}
		r.gatewayMux = gatewayruntime.NewServeMux(
			gatewayruntime.WithProtoErrorHandler(gatewayruntime.DefaultHTTPProtoErrorHandler),
			gatewayruntime.WithMarshalerOption("application/x-protobuf", &xProtobufMarshaler{}),
			gatewayruntime.WithMarshalerOption(gatewayruntime.MIMEWildcard, jsonpb),
		)
	}

	return r
}

func (r *otlpReceiver) startGRPCServer(cfg *configgrpc.GRPCServerSettings, host component.Host) error {
	r.logger.Info("Starting GRPC server on endpoint " + cfg.NetAddr.Endpoint)

	gln, err := cfg.ToListener()
	if err != nil {
		return err
	}
	r.shutdownWG.Add(1)
	go func() {
		defer r.shutdownWG.Done()

		if errGrpc := r.serverGRPC.Serve(gln); errGrpc != nil && errGrpc != grpc.ErrServerStopped {
			host.ReportFatalError(errGrpc)
		}
	}()
	return nil
}

func (r *otlpReceiver) startHTTPServer(cfg *confighttp.HTTPServerSettings, host component.Host) error {
	r.logger.Info("Starting HTTP server on endpoint " + cfg.Endpoint)
	var hln net.Listener
	hln, err := r.cfg.HTTP.ToListener()
	if err != nil {
		return err
	}
	r.shutdownWG.Add(1)
	go func() {
		defer r.shutdownWG.Done()

		if errHTTP := r.serverHTTP.Serve(hln); errHTTP != http.ErrServerClosed {
			host.ReportFatalError(errHTTP)
		}
	}()
	return nil
}

func (r *otlpReceiver) startProtocolServers(host component.Host) error {
	var err error
	if r.cfg.GRPC != nil {
		var opts []grpc.ServerOption
		opts, err = r.cfg.GRPC.ToServerOption(host.GetExtensions())
		if err != nil {
			return err
		}
		r.serverGRPC = grpc.NewServer(opts...)

		if r.traceReceiver != nil {
			collectortrace.RegisterTraceServiceServer(r.serverGRPC, r.traceReceiver)
		}

		if r.metricsReceiver != nil {
			collectormetrics.RegisterMetricsServiceServer(r.serverGRPC, r.metricsReceiver)
		}

		if r.logReceiver != nil {
			collectorlog.RegisterLogsServiceServer(r.serverGRPC, r.logReceiver)
		}

		err = r.startGRPCServer(r.cfg.GRPC, host)
		if err != nil {
			return err
		}
		if r.cfg.GRPC.NetAddr.Endpoint == defaultGRPCEndpoint {
			r.logger.Info("Setting up a second GRPC listener on legacy endpoint " + legacyGRPCEndpoint)

			// Copy the config.
			cfgLegacyGRPC := r.cfg.GRPC
			// And use the legacy endpoint.
			cfgLegacyGRPC.NetAddr.Endpoint = legacyGRPCEndpoint
			err = r.startGRPCServer(cfgLegacyGRPC, host)
			if err != nil {
				return err
			}
		}
	}
	if r.cfg.HTTP != nil {
		r.serverHTTP = r.cfg.HTTP.ToServer(
			r.gatewayMux,
			confighttp.WithErrorHandler(errorHandler),
		)
		err = r.startHTTPServer(r.cfg.HTTP, host)
		if err != nil {
			return err
		}
	}

	return err
}

// Start runs the trace receiver on the gRPC server. Currently
// it also enables the metrics receiver too.
func (r *otlpReceiver) Start(_ context.Context, host component.Host) error {
	return r.startProtocolServers(host)
}

// Shutdown is a method to turn off receiving.
func (r *otlpReceiver) Shutdown(ctx context.Context) error {
	var err error

	if r.serverHTTP != nil {
		err = r.serverHTTP.Shutdown(ctx)
	}

	if r.serverGRPC != nil {
		r.serverGRPC.GracefulStop()
	}

	r.shutdownWG.Wait()
	return err
}

func (r *otlpReceiver) registerTraceConsumer(ctx context.Context, tc consumer.Traces) error {
	if tc == nil {
		return componenterror.ErrNilNextConsumer
	}
	r.traceReceiver = trace.New(r.cfg.ID(), tc)
	if r.gatewayMux != nil {
		err := collectortrace.RegisterTraceServiceHandlerServer(ctx, r.gatewayMux, r.traceReceiver)
		if err != nil {
			return err
		}
		// Also register an alias handler. This fixes bug https://github.com/open-telemetry/opentelemetry-collector/issues/1968
		return collectortrace.RegisterTraceServiceHandlerServerAlias(ctx, r.gatewayMux, r.traceReceiver)
	}
	return nil
}

func (r *otlpReceiver) registerMetricsConsumer(ctx context.Context, mc consumer.Metrics) error {
	if mc == nil {
		return componenterror.ErrNilNextConsumer
	}
	r.metricsReceiver = metrics.New(r.cfg.ID(), mc)
	if r.gatewayMux != nil {
		return collectormetrics.RegisterMetricsServiceHandlerServer(ctx, r.gatewayMux, r.metricsReceiver)
	}
	return nil
}

func (r *otlpReceiver) registerLogsConsumer(ctx context.Context, tc consumer.Logs) error {
	if tc == nil {
		return componenterror.ErrNilNextConsumer
	}
	r.logReceiver = logs.New(r.cfg.ID(), tc)
	if r.gatewayMux != nil {
		return collectorlog.RegisterLogsServiceHandlerServer(ctx, r.gatewayMux, r.logReceiver)
	}
	return nil
}
