// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlpreceiver // import "go.opentelemetry.io/collector/receiver/otlpreceiver"

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/stats"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/internal/telemetry"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"go.opentelemetry.io/collector/pdata/pprofile/pprofileotlp"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/logs"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/metadata"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/metrics"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/profiles"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/trace"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
)

// otlpReceiver is the type that exposes Trace and Metrics reception.
type otlpReceiver struct {
	cfg        *Config
	serverGRPC *grpc.Server
	serverHTTP *http.Server

	nextTraces   consumer.Traces
	nextMetrics  consumer.Metrics
	nextLogs     consumer.Logs
	nextProfiles xconsumer.Profiles
	shutdownWG   sync.WaitGroup

	obsrepGRPC *receiverhelper.ObsReport
	obsrepHTTP *receiverhelper.ObsReport

	telemetryBuilder    *metadata.TelemetryBuilder
	activeConnsAttrGRPC metric.AddOption
	activeConnsAttrHTTP metric.AddOption

	settings *receiver.Settings
}

// newOtlpReceiver just creates the OpenTelemetry receiver services. It is the caller's
// responsibility to invoke the respective Start*Reception methods as well
// as the various Stop*Reception methods to end it.
func newOtlpReceiver(cfg *Config, set *receiver.Settings) (*otlpReceiver, error) {
	set.TelemetrySettings = telemetry.DropInjectedAttributes(set.TelemetrySettings, telemetry.SignalKey)
	set.Logger.Debug("created signal-agnostic logger")
	r := &otlpReceiver{
		cfg:          cfg,
		nextTraces:   nil,
		nextMetrics:  nil,
		nextLogs:     nil,
		nextProfiles: nil,
		settings:     set,
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

	r.telemetryBuilder, err = metadata.NewTelemetryBuilder(set.TelemetrySettings)
	if err != nil {
		return nil, err
	}
	r.activeConnsAttrGRPC = metric.WithAttributes(attribute.String("transport", "grpc"))
	r.activeConnsAttrHTTP = metric.WithAttributes(attribute.String("transport", "http"))

	return r, nil
}

// connCounterStatsHandler is a grpc/stats.Handler that tracks the number of
// currently open gRPC connections via HandleConn's ConnBegin/ConnEnd events.
// It is registered alongside the otelgrpc stats handler; grpc-go combines
// multiple registered stats handlers rather than overwriting them.
type connCounterStatsHandler struct {
	counter metric.Int64UpDownCounter
	attrs   metric.AddOption
}

func (connCounterStatsHandler) TagRPC(ctx context.Context, _ *stats.RPCTagInfo) context.Context {
	return ctx
}

func (connCounterStatsHandler) HandleRPC(context.Context, stats.RPCStats) {}

func (connCounterStatsHandler) TagConn(ctx context.Context, _ *stats.ConnTagInfo) context.Context {
	return ctx
}

func (h connCounterStatsHandler) HandleConn(ctx context.Context, s stats.ConnStats) {
	switch s.(type) {
	case *stats.ConnBegin:
		h.counter.Add(ctx, 1, h.attrs)
	case *stats.ConnEnd:
		h.counter.Add(ctx, -1, h.attrs)
	}
}

func (r *otlpReceiver) startGRPCServer(ctx context.Context, host component.Host) error {
	// If GRPC is not enabled, nothing to start.
	if !r.cfg.Protocols.GRPC.HasValue() {
		return nil
	}

	grpcCfg := r.cfg.Protocols.GRPC.Get()
	connCounter := connCounterStatsHandler{counter: r.telemetryBuilder.ReceiverOtlpActiveConnections, attrs: r.activeConnsAttrGRPC}
	var err error
	if r.serverGRPC, err = grpcCfg.ToServer(ctx, host.GetExtensions(), r.settings.TelemetrySettings, configgrpc.WithGrpcServerOption(grpc.StatsHandler(connCounter))); err != nil {
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

	if r.nextProfiles != nil {
		pprofileotlp.RegisterGRPCServer(r.serverGRPC, profiles.New(r.nextProfiles, r.obsrepGRPC))
	}

	var gln net.Listener
	if gln, err = grpcCfg.NetAddr.Listen(ctx); err != nil {
		return err
	}
	r.settings.Logger.Info("Starting GRPC server", zap.String("endpoint", gln.Addr().String()))

	r.shutdownWG.Go(func() {
		if errGrpc := r.serverGRPC.Serve(gln); errGrpc != nil && !errors.Is(errGrpc, grpc.ErrServerStopped) {
			componentstatus.ReportStatus(host, componentstatus.NewFatalErrorEvent(errGrpc))
		}
	})
	return nil
}

func (r *otlpReceiver) startHTTPServer(ctx context.Context, host component.Host) error {
	// If HTTP is not enabled, nothing to start.
	if !r.cfg.Protocols.HTTP.HasValue() {
		return nil
	}

	httpCfg := r.cfg.Protocols.HTTP.Get()
	httpMux := http.NewServeMux()
	if r.nextTraces != nil {
		httpTracesReceiver := trace.New(r.nextTraces, r.obsrepHTTP)
		httpMux.HandleFunc(string(httpCfg.TracesURLPath), func(resp http.ResponseWriter, req *http.Request) {
			handleTraces(resp, req, httpTracesReceiver)
		})
	}

	if r.nextMetrics != nil {
		httpMetricsReceiver := metrics.New(r.nextMetrics, r.obsrepHTTP)
		httpMux.HandleFunc(string(httpCfg.MetricsURLPath), func(resp http.ResponseWriter, req *http.Request) {
			handleMetrics(resp, req, httpMetricsReceiver)
		})
	}

	if r.nextLogs != nil {
		httpLogsReceiver := logs.New(r.nextLogs, r.obsrepHTTP)
		httpMux.HandleFunc(string(httpCfg.LogsURLPath), func(resp http.ResponseWriter, req *http.Request) {
			handleLogs(resp, req, httpLogsReceiver)
		})
	}

	if r.nextProfiles != nil {
		httpProfilesReceiver := profiles.New(r.nextProfiles, r.obsrepHTTP)
		httpMux.HandleFunc(defaultProfilesURLPath, func(resp http.ResponseWriter, req *http.Request) {
			handleProfiles(resp, req, httpProfilesReceiver)
		})
	}

	connStateCallback := func(_ net.Conn, state http.ConnState) {
		switch state {
		case http.StateNew:
			r.telemetryBuilder.ReceiverOtlpActiveConnections.Add(context.Background(), 1, r.activeConnsAttrHTTP)
		case http.StateClosed, http.StateHijacked:
			r.telemetryBuilder.ReceiverOtlpActiveConnections.Add(context.Background(), -1, r.activeConnsAttrHTTP)
		}
	}

	var err error
	if r.serverHTTP, err = httpCfg.ServerConfig.ToServer(ctx, host.GetExtensions(), r.settings.TelemetrySettings, httpMux, confighttp.WithErrorHandler(errorHandler), confighttp.WithConnStateCallback(connStateCallback)); err != nil {
		return err
	}

	var hln net.Listener
	if hln, err = httpCfg.ServerConfig.ToListener(ctx); err != nil {
		return err
	}
	r.settings.Logger.Info("Starting HTTP server", zap.String("endpoint", hln.Addr().String()))

	r.shutdownWG.Go(func() {
		if errHTTP := r.serverHTTP.Serve(hln); errHTTP != nil && !errors.Is(errHTTP, http.ErrServerClosed) {
			componentstatus.ReportStatus(host, componentstatus.NewFatalErrorEvent(errHTTP))
		}
	})
	return nil
}

// Start runs the trace receiver on the gRPC server. Currently
// it also enables the metrics receiver too.
func (r *otlpReceiver) Start(ctx context.Context, host component.Host) error {
	if err := r.startGRPCServer(ctx, host); err != nil {
		return err
	}
	if err := r.startHTTPServer(ctx, host); err != nil {
		// It's possible that a valid GRPC server configuration was specified,
		// but an invalid HTTP configuration. If that's the case, the successfully
		// started GRPC server must be shutdown to ensure no goroutines are leaked.
		return errors.Join(err, r.Shutdown(ctx))
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

func (r *otlpReceiver) registerTraceConsumer(tc consumer.Traces) {
	r.nextTraces = tc
}

func (r *otlpReceiver) registerMetricsConsumer(mc consumer.Metrics) {
	r.nextMetrics = mc
}

func (r *otlpReceiver) registerLogsConsumer(lc consumer.Logs) {
	r.nextLogs = lc
}

func (r *otlpReceiver) registerProfilesConsumer(tc xconsumer.Profiles) {
	r.nextProfiles = tc
}
