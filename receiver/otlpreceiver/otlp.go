// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlpreceiver // import "go.opentelemetry.io/collector/receiver/otlpreceiver"

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
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
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/metrics"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/profiles"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/trace"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.opentelemetry.io/collector/receiver/receiverhelper/xreceiverhelper"
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

	// Profiles are observed separately, as their telemetry is reported per
	// sample type.
	obsrepProfilesGRPC *xreceiverhelper.ObsReport
	obsrepProfilesHTTP *xreceiverhelper.ObsReport

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
	r.obsrepProfilesGRPC, err = xreceiverhelper.NewObsReport(receiverhelper.ObsReportSettings{
		ReceiverID:             set.ID,
		Transport:              "grpc",
		ReceiverCreateSettings: *set,
	})
	if err != nil {
		return nil, err
	}
	r.obsrepProfilesHTTP, err = xreceiverhelper.NewObsReport(receiverhelper.ObsReportSettings{
		ReceiverID:             set.ID,
		Transport:              "http",
		ReceiverCreateSettings: *set,
	})
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (r *otlpReceiver) startGRPCServer(ctx context.Context, host component.Host) error {
	// If GRPC is not enabled, nothing to start.
	if !r.cfg.Protocols.GRPC.HasValue() {
		return nil
	}

	grpcCfg := r.cfg.Protocols.GRPC.Get()
	var err error
	if r.serverGRPC, err = grpcCfg.ToServer(ctx, host.GetExtensions(), r.settings.TelemetrySettings); err != nil {
		return err
	}

	// Register through the server's registrar so that, when authentication is configured,
	// requests are authenticated before they are decompressed and decoded.
	registrar, err := grpcCfg.ServiceRegistrar(ctx, host.GetExtensions(), r.serverGRPC)
	if err != nil {
		return err
	}

	if r.nextTraces != nil {
		ptraceotlp.RegisterGRPCService(registrar, trace.New(r.nextTraces, r.obsrepGRPC))
	}

	if r.nextMetrics != nil {
		pmetricotlp.RegisterGRPCService(registrar, metrics.New(r.nextMetrics, r.obsrepGRPC))
	}

	if r.nextLogs != nil {
		plogotlp.RegisterGRPCService(registrar, logs.New(r.nextLogs, r.obsrepGRPC))
	}

	if r.nextProfiles != nil {
		pprofileotlp.RegisterGRPCService(registrar, profiles.New(r.nextProfiles, r.obsrepProfilesGRPC))
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
		httpProfilesReceiver := profiles.New(r.nextProfiles, r.obsrepProfilesHTTP)
		httpMux.HandleFunc(defaultProfilesURLPath, func(resp http.ResponseWriter, req *http.Request) {
			handleProfiles(resp, req, httpProfilesReceiver)
		})
	}

	var err error
	if r.serverHTTP, err = httpCfg.ServerConfig.ToServer(ctx, host.GetExtensions(), r.settings.TelemetrySettings, httpMux, confighttp.WithErrorHandler(errorHandler)); err != nil {
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

// Start runs the OTLP receiver's gRPC and HTTP servers, serving
// traces, metrics, logs (and profiles when enabled).
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

// shutdownGracePeriod bounds how long Shutdown waits for in-flight requests to
// complete when the caller's context carries no deadline. A request is in flight
// from the moment its headers arrive, before it has been authenticated, so without
// a bound a peer that opens a request and never completes it would block the
// collector's shutdown or reload for as long as it keeps the connection open.
//
// It is added on top of the longest time the server itself allows a request to
// take (for HTTP, the configured read and write timeouts), so that raising those
// limits does not cause legitimate requests to be cut off during shutdown.
const shutdownGracePeriod = 30 * time.Second

// shutdownContext returns ctx unchanged when it already carries a deadline,
// otherwise a context that expires after limit plus shutdownGracePeriod.
func shutdownContext(ctx context.Context, limit time.Duration) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, limit+shutdownGracePeriod)
}

// Shutdown is a method to turn off receiving.
func (r *otlpReceiver) Shutdown(ctx context.Context) error {
	var err error

	if r.serverHTTP != nil {
		// A request can legitimately last up to the larger of the read and write
		// timeouts (zero means no limit, which leaves just the grace period).
		var limit time.Duration
		if httpCfg := r.cfg.Protocols.HTTP.Get(); httpCfg != nil {
			limit = max(httpCfg.ServerConfig.WriteTimeout, httpCfg.ServerConfig.ReadTimeout)
		}
		httpCtx, cancel := shutdownContext(ctx, limit)
		defer cancel()
		err = r.serverHTTP.Shutdown(httpCtx)
		if httpCtx.Err() != nil {
			// Shutdown returned without draining every connection; close the
			// remaining ones instead of leaving the collector waiting on them.
			r.settings.Logger.Warn("HTTP server did not drain in time, closing open connections")
			err = errors.Join(err, r.serverHTTP.Close())
		}
	}

	if r.serverGRPC != nil {
		// gRPC has no per-request server-side timeout, so only the grace period applies.
		grpcCtx, cancel := shutdownContext(ctx, 0)
		defer cancel()
		stopped := make(chan struct{})
		go func() {
			defer close(stopped)
			r.serverGRPC.GracefulStop()
		}()
		select {
		case <-stopped:
		case <-grpcCtx.Done():
			// Stop is the documented way to abort a GracefulStop that is taking
			// too long: it closes the remaining connections and cancels their RPCs.
			r.settings.Logger.Warn("gRPC server did not drain in time, closing open connections")
			r.serverGRPC.Stop()
			<-stopped
		}
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
