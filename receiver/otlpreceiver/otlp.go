// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlpreceiver // import "go.opentelemetry.io/collector/receiver/otlpreceiver"

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/logs"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/metrics"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/trace"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
)

// otlpReceiver is the type that exposes Trace and Metrics reception.
type otlpReceiver struct {
	cfg        *Config
	serverGRPC *grpc.Server
	serverHTTP *http.Server

	nextTraces  consumer.Traces
	nextMetrics consumer.Metrics
	nextLogs    consumer.Logs
	shutdownWG  sync.WaitGroup

	obsrepGRPC *receiverhelper.ObsReport
	obsrepHTTP *receiverhelper.ObsReport

	settings *receiver.CreateSettings
}

// newOtlpReceiver just creates the OpenTelemetry receiver services. It is the caller's
// responsibility to invoke the respective Start*Reception methods as well
// as the various Stop*Reception methods to end it.
func newOtlpReceiver(cfg *Config, set *receiver.CreateSettings) (*otlpReceiver, error) {
	r := &otlpReceiver{
		cfg:         cfg,
		nextTraces:  nil,
		nextMetrics: nil,
		nextLogs:    nil,
		settings:    set,
	}

	var err error
	r.obsrepGRPC, err = receiverhelper.NewObsReport(receiverhelper.ObsReportSettings{
		ReceiverID:             set.ID,
		Transport:              "grpc",
		ReceiverCreateSettings: *set,
	})
	if err != nil {
		return nil, err
	}
	r.obsrepHTTP, err = receiverhelper.NewObsReport(receiverhelper.ObsReportSettings{
		ReceiverID:             set.ID,
		Transport:              "http",
		ReceiverCreateSettings: *set,
	})
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (r *otlpReceiver) startGRPCServer(host component.Host) error {
	// If GRPC is not enabled, nothing to start.
	if r.cfg.GRPC == nil {
		return nil
	}

	var err error
	if r.serverGRPC, err = r.cfg.GRPC.ToServer(host, r.settings.TelemetrySettings); err != nil {
		return err
	}

	if r.nextTraces != nil {
		ptraceotlp.RegisterGRPCServer(r.serverGRPC, trace.New(r.nextTraces, r.obsrepGRPC))
	}

	if r.nextMetrics != nil {
		pmetricotlp.RegisterGRPCServer(r.serverGRPC, metrics.New(r.nextMetrics, r.obsrepGRPC))
	}

	if r.nextLogs != nil {
		plogotlp.RegisterGRPCServer(r.serverGRPC, logs.New(r.nextLogs, r.obsrepGRPC))
	}

	r.settings.Logger.Info("Starting GRPC server", zap.String("endpoint", r.cfg.GRPC.NetAddr.Endpoint))
	var gln net.Listener
	if gln, err = r.cfg.GRPC.ToListener(); err != nil {
		return err
	}

	r.shutdownWG.Add(1)
	go func() {
		defer r.shutdownWG.Done()

		if errGrpc := r.serverGRPC.Serve(gln); errGrpc != nil && !errors.Is(errGrpc, grpc.ErrServerStopped) {
			host.ReportFatalError(errGrpc)
		}
	}()
	return nil
}

func (r *otlpReceiver) startHTTPServer(host component.Host) error {
	// If HTTP is not enabled, nothing to start.
	if r.cfg.HTTP == nil {
		return nil
	}

	httpMux := http.NewServeMux()
	if r.nextTraces != nil {
		httpTracesReceiver := trace.New(r.nextTraces, r.obsrepHTTP)
		httpMux.HandleFunc(r.cfg.HTTP.TracesURLPath, func(resp http.ResponseWriter, req *http.Request) {
			handleTraces(resp, req, httpTracesReceiver)
		})
	}

	if r.nextMetrics != nil {
		httpMetricsReceiver := metrics.New(r.nextMetrics, r.obsrepHTTP)
		httpMux.HandleFunc(r.cfg.HTTP.MetricsURLPath, func(resp http.ResponseWriter, req *http.Request) {
			handleMetrics(resp, req, httpMetricsReceiver)
		})
	}

	if r.nextLogs != nil {
		httpLogsReceiver := logs.New(r.nextLogs, r.obsrepHTTP)
		httpMux.HandleFunc(r.cfg.HTTP.LogsURLPath, func(resp http.ResponseWriter, req *http.Request) {
			handleLogs(resp, req, httpLogsReceiver)
		})
	}

	var err error
	if r.serverHTTP, err = r.cfg.HTTP.ToServer(host, r.settings.TelemetrySettings, httpMux, confighttp.WithErrorHandler(errorHandler)); err != nil {
		return err
	}

	r.settings.Logger.Info("Starting HTTP server", zap.String("endpoint", r.cfg.HTTP.HTTPServerSettings.Endpoint))
	var hln net.Listener
	if hln, err = r.cfg.HTTP.HTTPServerSettings.ToListener(); err != nil {
		return err
	}

	r.shutdownWG.Add(1)
	go func() {
		defer r.shutdownWG.Done()

		if errHTTP := r.serverHTTP.Serve(hln); errHTTP != nil && !errors.Is(errHTTP, http.ErrServerClosed) {
			host.ReportFatalError(errHTTP)
		}
	}()
	return nil
}

// Start runs the trace receiver on the gRPC server. Currently
// it also enables the metrics receiver too.
func (r *otlpReceiver) Start(_ context.Context, host component.Host) error {
	if err := r.startGRPCServer(host); err != nil {
		return err
	}
	if err := r.startHTTPServer(host); err != nil {
		return err
	}

	return nil
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

func (r *otlpReceiver) registerTraceConsumer(tc consumer.Traces) error {
	if tc == nil {
		return component.ErrNilNextConsumer
	}
	r.nextTraces = tc
	return nil
}

func (r *otlpReceiver) registerMetricsConsumer(mc consumer.Metrics) error {
	if mc == nil {
		return component.ErrNilNextConsumer
	}
	r.nextMetrics = mc
	return nil
}

func (r *otlpReceiver) registerLogsConsumer(lc consumer.Logs) error {
	if lc == nil {
		return component.ErrNilNextConsumer
	}
	r.nextLogs = lc
	return nil
}
