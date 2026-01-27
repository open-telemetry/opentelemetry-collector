// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlpexporter // import "go.opentelemetry.io/collector/exporter/otlpexporter"

import (
	"context"
	"fmt"
	"runtime"
	"sync/atomic"

	"go.uber.org/zap"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/internal/statusutil"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/pprofile/pprofileotlp"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
)

type baseExporter struct {
	// Input configuration.
	config *Config

	// gRPC clients and connections pool
	// When connection pool size > 1, we maintain multiple connections and round-robin across them
	traceExporters   []ptraceotlp.GRPCClient
	metricExporters  []pmetricotlp.GRPCClient
	logExporters     []plogotlp.GRPCClient
	profileExporters []pprofileotlp.GRPCClient
	clientConns      []*grpc.ClientConn
	
	// Round-robin counter for load balancing across connections
	roundRobinCounter atomic.Uint32
	
	metadata    metadata.MD
	callOptions []grpc.CallOption

	settings component.TelemetrySettings

	// Default user-agent header.
	userAgent string
}

func newExporter(cfg component.Config, set exporter.Settings) *baseExporter {
	oCfg := cfg.(*Config)

	userAgent := fmt.Sprintf("%s/%s (%s/%s)",
		set.BuildInfo.Description, set.BuildInfo.Version, runtime.GOOS, runtime.GOARCH)

	return &baseExporter{config: oCfg, settings: set.TelemetrySettings, userAgent: userAgent}
}

// start actually creates the gRPC connection(s). The client construction is deferred till this point as this
// is the only place we get hold of Extensions which are required to construct auth round tripper.
func (e *baseExporter) start(ctx context.Context, host component.Host) (err error) {
	agentOpt := configgrpc.WithGrpcDialOption(grpc.WithUserAgent(e.userAgent))
	
	// Determine pool size: default to 1 if not configured or 0
	poolSize := e.config.ConnectionPoolSize
	if poolSize <= 0 {
		poolSize = 1
	}
	
	// Initialize slices for connection pool
	e.clientConns = make([]*grpc.ClientConn, poolSize)
	e.traceExporters = make([]ptraceotlp.GRPCClient, poolSize)
	e.metricExporters = make([]pmetricotlp.GRPCClient, poolSize)
	e.logExporters = make([]plogotlp.GRPCClient, poolSize)
	e.profileExporters = make([]pprofileotlp.GRPCClient, poolSize)
	
	// Create multiple connections for the pool
	for i := 0; i < poolSize; i++ {
		if e.clientConns[i], err = e.config.ClientConfig.ToClientConn(ctx, host.GetExtensions(), e.settings, agentOpt); err != nil {
			// Close any connections created before the error
			for j := 0; j < i; j++ {
				if e.clientConns[j] != nil {
					_ = e.clientConns[j].Close()
				}
			}
			return err
		}
		e.traceExporters[i] = ptraceotlp.NewGRPCClient(e.clientConns[i])
		e.metricExporters[i] = pmetricotlp.NewGRPCClient(e.clientConns[i])
		e.logExporters[i] = plogotlp.NewGRPCClient(e.clientConns[i])
		e.profileExporters[i] = pprofileotlp.NewGRPCClient(e.clientConns[i])
	}
	
	if poolSize > 1 {
		e.settings.Logger.Info("OTLP exporter connection pool created",
			zap.Int("pool_size", poolSize),
			zap.String("endpoint", e.config.ClientConfig.Endpoint),
		)
	}
	
	headers := map[string]string{}
	for k, v := range e.config.ClientConfig.Headers.Iter {
		headers[k] = string(v)
	}
	e.metadata = metadata.New(headers)
	e.callOptions = []grpc.CallOption{
		grpc.WaitForReady(e.config.ClientConfig.WaitForReady),
	}

	return err
}

func (e *baseExporter) shutdown(context.Context) error {
	var lastErr error
	for _, conn := range e.clientConns {
		if conn != nil {
			if err := conn.Close(); err != nil {
				lastErr = err
			}
		}
	}
	return lastErr
}

// getNextExporterIndex returns the next index for round-robin load balancing across connections
func (e *baseExporter) getNextExporterIndex() int {
	if len(e.clientConns) == 1 {
		return 0
	}
	counter := e.roundRobinCounter.Add(1)
	return int(counter % uint32(len(e.clientConns)))
}

func (e *baseExporter) pushTraces(ctx context.Context, td ptrace.Traces) error {
	idx := e.getNextExporterIndex()
	req := ptraceotlp.NewExportRequestFromTraces(td)
	resp, respErr := e.traceExporters[idx].Export(ctx, req, e.callOptions...)
	if err := processError(respErr); err != nil {
		return err
	}
	partialSuccess := resp.PartialSuccess()
	if partialSuccess.ErrorMessage() != "" || partialSuccess.RejectedSpans() != 0 {
		e.settings.Logger.Warn("Partial success response",
			zap.String("message", resp.PartialSuccess().ErrorMessage()),
			zap.Int64("dropped_spans", resp.PartialSuccess().RejectedSpans()),
		)
	}
	return nil
}

func (e *baseExporter) pushMetrics(ctx context.Context, md pmetric.Metrics) error {
	idx := e.getNextExporterIndex()
	req := pmetricotlp.NewExportRequestFromMetrics(md)
	resp, respErr := e.metricExporters[idx].Export(ctx, req, e.callOptions...)
	if err := processError(respErr); err != nil {
		return err
	}
	partialSuccess := resp.PartialSuccess()
	if partialSuccess.ErrorMessage() != "" || partialSuccess.RejectedDataPoints() != 0 {
		e.settings.Logger.Warn("Partial success response",
			zap.String("message", resp.PartialSuccess().ErrorMessage()),
			zap.Int64("dropped_data_points", resp.PartialSuccess().RejectedDataPoints()),
		)
	}
	return nil
}

func (e *baseExporter) pushLogs(ctx context.Context, ld plog.Logs) error {
	idx := e.getNextExporterIndex()
	req := plogotlp.NewExportRequestFromLogs(ld)
	resp, respErr := e.logExporters[idx].Export(ctx, req, e.callOptions...)
	if err := processError(respErr); err != nil {
		return err
	}
	partialSuccess := resp.PartialSuccess()
	if partialSuccess.ErrorMessage() != "" || partialSuccess.RejectedLogRecords() != 0 {
		e.settings.Logger.Warn("Partial success response",
			zap.String("message", resp.PartialSuccess().ErrorMessage()),
			zap.Int64("dropped_log_records", resp.PartialSuccess().RejectedLogRecords()),
		)
	}
	return nil
}

func (e *baseExporter) pushProfiles(ctx context.Context, td pprofile.Profiles) error {
	idx := e.getNextExporterIndex()
	req := pprofileotlp.NewExportRequestFromProfiles(td)
	resp, respErr := e.profileExporters[idx].Export(ctx, req, e.callOptions...)
	if err := processError(respErr); err != nil {
		return err
	}
	partialSuccess := resp.PartialSuccess()
	if partialSuccess.ErrorMessage() != "" || partialSuccess.RejectedProfiles() != 0 {
		e.settings.Logger.Warn("Partial success response",
			zap.String("message", resp.PartialSuccess().ErrorMessage()),
			zap.Int64("dropped_profiles", resp.PartialSuccess().RejectedProfiles()),
		)
	}
	return nil
}

func processError(err error) error {
	if err == nil {
		// Request is successful, we are done.
		return nil
	}

	// We have an error, check gRPC status code.
	st := status.Convert(err)
	if st.Code() == codes.OK {
		// Not really an error, still success.
		return nil
	}

	// Now, this is a real error.
	retryInfo := statusutil.GetRetryInfo(st)

	if !shouldRetry(st.Code(), retryInfo) {
		// It is not a retryable error, we should not retry.
		return consumererror.NewPermanent(err)
	}

	// Check if server returned throttling information.
	throttleDuration := retryInfo.GetRetryDelay().AsDuration()
	if throttleDuration != 0 {
		// We are throttled. Wait before retrying as requested by the server.
		return exporterhelper.NewThrottleRetry(err, throttleDuration)
	}

	// Need to retry.
	return err
}

func shouldRetry(code codes.Code, retryInfo *errdetails.RetryInfo) bool {
	switch code {
	case codes.Canceled,
		codes.DeadlineExceeded,
		codes.Aborted,
		codes.OutOfRange,
		codes.Unavailable,
		codes.DataLoss:
		// These are retryable errors.
		return true
	case codes.ResourceExhausted:
		// Retry only if RetryInfo was supplied by the server.
		// This indicates that the server can still recover from resource exhaustion.
		return retryInfo != nil
	}
	// Don't retry on any other code.
	return false
}
