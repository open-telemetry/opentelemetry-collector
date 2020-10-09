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

package otlpexporter

import (
	"context"
	"errors"
	"fmt"
	"time"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/internal"
	otlplogs "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/collector/logs/v1"
	otlpmetrics "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/collector/metrics/v1"
	otlptrace "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/collector/trace/v1"
)

type exporterImp struct {
	// Input configuration.
	config *Config
	w      *grpcSender
}

var (
	errPermanentError = consumererror.Permanent(errors.New("fatal error sending to server"))
)

// Crete new exporter and start it. The exporter will begin connecting but
// this function may return before the connection is established.
func newExporter(cfg configmodels.Exporter) (*exporterImp, error) {
	oCfg := cfg.(*Config)

	if oCfg.Endpoint == "" {
		return nil, errors.New("OTLP exporter config requires an Endpoint")
	}

	e := &exporterImp{}
	e.config = oCfg
	w, err := newGrpcSender(oCfg)
	if err != nil {
		return nil, err
	}
	e.w = w
	return e, nil
}

func (e *exporterImp) shutdown(context.Context) error {
	return e.w.stop()
}

func (e *exporterImp) pushTraceData(ctx context.Context, td pdata.Traces) (int, error) {
	request := &otlptrace.ExportTraceServiceRequest{
		ResourceSpans: pdata.TracesToOtlp(td),
	}
	err := e.w.exportTrace(ctx, request)

	if err != nil {
		return td.SpanCount(), fmt.Errorf("failed to push trace data via OTLP exporter: %w", err)
	}
	return 0, nil
}

func (e *exporterImp) pushMetricsData(ctx context.Context, md pdata.Metrics) (int, error) {
	request := &otlpmetrics.ExportMetricsServiceRequest{
		ResourceMetrics: pdata.MetricsToOtlp(md),
	}
	err := e.w.exportMetrics(ctx, request)

	if err != nil {
		return md.MetricCount(), fmt.Errorf("failed to push metrics data via OTLP exporter: %w", err)
	}
	return 0, nil
}

func (e *exporterImp) pushLogData(ctx context.Context, logs pdata.Logs) (int, error) {
	request := &otlplogs.ExportLogsServiceRequest{
		ResourceLogs: internal.LogsToOtlp(logs.InternalRep()),
	}
	err := e.w.exportLogs(ctx, request)

	if err != nil {
		return logs.LogRecordCount(), fmt.Errorf("failed to push log data via OTLP exporter: %w", err)
	}
	return 0, nil
}

type grpcSender struct {
	// gRPC clients and connection.
	traceExporter  otlptrace.TraceServiceClient
	metricExporter otlpmetrics.MetricsServiceClient
	logExporter    otlplogs.LogsServiceClient
	grpcClientConn *grpc.ClientConn
	metadata       metadata.MD
	waitForReady   bool
}

func newGrpcSender(config *Config) (*grpcSender, error) {
	dialOpts, err := config.GRPCClientSettings.ToDialOptions()
	if err != nil {
		return nil, err
	}

	var clientConn *grpc.ClientConn
	if clientConn, err = grpc.Dial(config.GRPCClientSettings.Endpoint, dialOpts...); err != nil {
		return nil, err
	}

	gs := &grpcSender{
		traceExporter:  otlptrace.NewTraceServiceClient(clientConn),
		metricExporter: otlpmetrics.NewMetricsServiceClient(clientConn),
		logExporter:    otlplogs.NewLogsServiceClient(clientConn),
		grpcClientConn: clientConn,
		metadata:       metadata.New(config.GRPCClientSettings.Headers),
		waitForReady:   config.GRPCClientSettings.WaitForReady,
	}
	return gs, nil
}

func (gs *grpcSender) stop() error {
	return gs.grpcClientConn.Close()
}

func (gs *grpcSender) exportTrace(ctx context.Context, request *otlptrace.ExportTraceServiceRequest) error {
	_, err := gs.traceExporter.Export(gs.enhanceContext(ctx), request, grpc.WaitForReady(gs.waitForReady))
	return processError(err)
}

func (gs *grpcSender) exportMetrics(ctx context.Context, request *otlpmetrics.ExportMetricsServiceRequest) error {
	_, err := gs.metricExporter.Export(gs.enhanceContext(ctx), request, grpc.WaitForReady(gs.waitForReady))
	return processError(err)
}

func (gs *grpcSender) exportLogs(ctx context.Context, request *otlplogs.ExportLogsServiceRequest) error {
	_, err := gs.logExporter.Export(gs.enhanceContext(ctx), request, grpc.WaitForReady(gs.waitForReady))
	return processError(err)
}

func (gs *grpcSender) enhanceContext(ctx context.Context) context.Context {
	if gs.metadata.Len() > 0 {
		return metadata.NewOutgoingContext(ctx, gs.metadata)
	}
	return ctx
}

// Send a trace or metrics request to the server. "perform" function is expected to make
// the actual gRPC unary call that sends the request. This function implements the
// common OTLP logic around request handling such as retries and throttling.
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

	// Now, this is this a real error.

	if !shouldRetry(st.Code()) {
		// It is not a retryable error, we should not retry.
		return errPermanentError
	}

	// Need to retry.

	// Check if server returned throttling information.
	throttleDuration := getThrottleDuration(st)
	if throttleDuration != 0 {
		return exporterhelper.NewThrottleRetry(err, throttleDuration)
	}

	return err
}

func shouldRetry(code codes.Code) bool {
	switch code {
	case codes.OK:
		// Success. This function should not be called for this code, the best we
		// can do is tell the caller not to retry.
		return false

	case codes.Canceled,
		codes.DeadlineExceeded,
		codes.PermissionDenied,
		codes.Unauthenticated,
		codes.ResourceExhausted,
		codes.Aborted,
		codes.OutOfRange,
		codes.Unavailable,
		codes.DataLoss:
		// These are retryable errors.
		return true

	case codes.Unknown,
		codes.InvalidArgument,
		codes.NotFound,
		codes.AlreadyExists,
		codes.FailedPrecondition,
		codes.Unimplemented,
		codes.Internal:
		// These are fatal errors, don't retry.
		return false

	default:
		// Don't retry on unknown codes.
		return false
	}
}

func getThrottleDuration(status *status.Status) time.Duration {
	// See if throttling information is available.
	for _, detail := range status.Details() {
		if t, ok := detail.(*errdetails.RetryInfo); ok {
			if t.RetryDelay.Seconds > 0 || t.RetryDelay.Nanos > 0 {
				// We are throttled. Wait before retrying as requested by the server.
				return time.Duration(t.RetryDelay.Seconds)*time.Second + time.Duration(t.RetryDelay.Nanos)*time.Nanosecond
			}
			return 0
		}
	}
	return 0
}
