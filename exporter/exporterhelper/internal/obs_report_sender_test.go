// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	grpccodes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/experr"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/metadatatest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
	"go.opentelemetry.io/collector/pipeline"
)

var (
	exporterID = component.MustNewID("fakeExporter")

	errFake = errors.New("errFake")
)

func TestExportTraceFailureAttributesRetryExhausted(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	obsrep, err := newObsReportSender(
		exporter.Settings{ID: exporterID, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		pipeline.SignalTraces,
		sender.NewSender(func(context.Context, request.Request) error {
			return NewRetryExhaustedErr(errFake)
		}),
	)
	require.NoError(t, err)

	req := &requesttest.FakeRequest{Items: 7}
	sendErr := obsrep.Send(context.Background(), req)
	require.Error(t, sendErr)

	wantAttrs := attribute.NewSet(
		attribute.String("exporter", exporterID.String()),
		attribute.String(string(semconv.ErrorTypeKey), "Unknown"),
		attribute.Bool(FailurePermanentKey, false),
		attribute.Bool(FailureRetriesExhaustedKey, true),
	)

	metadatatest.AssertEqualExporterSendFailedSpans(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: wantAttrs,
				Value:      int64(req.Items),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
}

func TestExportTraceFailureAttributesPermanentError(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	obsrep, err := newObsReportSender(
		exporter.Settings{ID: exporterID, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		pipeline.SignalTraces,
		sender.NewSender(func(context.Context, request.Request) error {
			return consumererror.NewPermanent(errors.New("bad data"))
		}),
	)
	require.NoError(t, err)

	req := &requesttest.FakeRequest{Items: 5}
	sendErr := obsrep.Send(context.Background(), req)
	require.Error(t, sendErr)

	wantAttrs := attribute.NewSet(
		attribute.String("exporter", exporterID.String()),
		attribute.String(string(semconv.ErrorTypeKey), "Unknown"),
		attribute.Bool(FailurePermanentKey, true),
		attribute.Bool(FailureRetriesExhaustedKey, false),
	)

	metadatatest.AssertEqualExporterSendFailedSpans(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: wantAttrs,
				Value:      int64(req.Items),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
}

func TestExportTraceFailureAttributesShutdownError(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	obsrep, err := newObsReportSender(
		exporter.Settings{ID: exporterID, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		pipeline.SignalTraces,
		sender.NewSender(func(context.Context, request.Request) error {
			return experr.NewShutdownErr(errors.New("shutting down"))
		}),
	)
	require.NoError(t, err)

	req := &requesttest.FakeRequest{Items: 3}
	sendErr := obsrep.Send(context.Background(), req)
	require.Error(t, sendErr)

	wantAttrs := attribute.NewSet(
		attribute.String("exporter", exporterID.String()),
		attribute.String(string(semconv.ErrorTypeKey), "Shutdown"),
		attribute.Bool(FailurePermanentKey, false),
		attribute.Bool(FailureRetriesExhaustedKey, false),
	)

	metadatatest.AssertEqualExporterSendFailedSpans(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: wantAttrs,
				Value:      int64(req.Items),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
}

func TestExportTraceFailureAttributesContextCanceled(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	obsrep, err := newObsReportSender(
		exporter.Settings{ID: exporterID, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		pipeline.SignalTraces,
		sender.NewSender(func(context.Context, request.Request) error {
			return context.Canceled
		}),
	)
	require.NoError(t, err)

	req := &requesttest.FakeRequest{Items: 2}
	sendErr := obsrep.Send(ctx, req)
	require.Error(t, sendErr)

	wantAttrs := attribute.NewSet(
		attribute.String("exporter", exporterID.String()),
		attribute.String(string(semconv.ErrorTypeKey), "Canceled"),
		attribute.Bool(FailurePermanentKey, false),
		attribute.Bool(FailureRetriesExhaustedKey, false),
	)

	metadatatest.AssertEqualExporterSendFailedSpans(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: wantAttrs,
				Value:      int64(req.Items),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
}

func TestExportTraceFailureAttributesContextDeadlineExceeded(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	obsrep, err := newObsReportSender(
		exporter.Settings{ID: exporterID, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		pipeline.SignalTraces,
		sender.NewSender(func(context.Context, request.Request) error {
			return context.DeadlineExceeded
		}),
	)
	require.NoError(t, err)

	req := &requesttest.FakeRequest{Items: 4}
	sendErr := obsrep.Send(context.Background(), req)
	require.Error(t, sendErr)

	wantAttrs := attribute.NewSet(
		attribute.String("exporter", exporterID.String()),
		attribute.String(string(semconv.ErrorTypeKey), "DeadlineExceeded"),
		attribute.Bool(FailurePermanentKey, false),
		attribute.Bool(FailureRetriesExhaustedKey, false),
	)

	metadatatest.AssertEqualExporterSendFailedSpans(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: wantAttrs,
				Value:      int64(req.Items),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
}

func TestExportTraceFailureAttributesUnknownError(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	obsrep, err := newObsReportSender(
		exporter.Settings{ID: exporterID, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		pipeline.SignalTraces,
		sender.NewSender(func(context.Context, request.Request) error {
			return errFake
		}),
	)
	require.NoError(t, err)

	req := &requesttest.FakeRequest{Items: 8}
	sendErr := obsrep.Send(context.Background(), req)
	require.Error(t, sendErr)

	wantAttrs := attribute.NewSet(
		attribute.String("exporter", exporterID.String()),
		attribute.String(string(semconv.ErrorTypeKey), "Unknown"),
		attribute.Bool(FailurePermanentKey, false),
		attribute.Bool(FailureRetriesExhaustedKey, false),
	)

	metadatatest.AssertEqualExporterSendFailedSpans(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: wantAttrs,
				Value:      int64(req.Items),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
}

func TestExportTraceFailureAttributesGRPCError(t *testing.T) {
	tests := []struct {
		name         string
		grpcCode     grpccodes.Code
		expectedType string
		isPermanent  bool
	}{
		{
			name:         "Unavailable",
			grpcCode:     grpccodes.Unavailable,
			expectedType: "Unavailable",
			isPermanent:  false,
		},
		{
			name:         "ResourceExhausted",
			grpcCode:     grpccodes.ResourceExhausted,
			expectedType: "ResourceExhausted",
			isPermanent:  false,
		},
		{
			name:         "DataLoss",
			grpcCode:     grpccodes.DataLoss,
			expectedType: "DataLoss",
			isPermanent:  false,
		},
		{
			name:         "InvalidArgument",
			grpcCode:     grpccodes.InvalidArgument,
			expectedType: "InvalidArgument",
			isPermanent:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			telemetry := componenttest.NewTelemetry()
			t.Cleanup(func() { require.NoError(t, telemetry.Shutdown(context.Background())) })

			grpcErr := status.Error(tt.grpcCode, "test error")
			obsrep, err := newObsReportSender(
				exporter.Settings{ID: exporterID, TelemetrySettings: telemetry.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
				pipeline.SignalTraces,
				sender.NewSender(func(context.Context, request.Request) error {
					return grpcErr
				}),
			)
			require.NoError(t, err)

			req := &requesttest.FakeRequest{Items: 10}
			sendErr := obsrep.Send(context.Background(), req)
			require.Error(t, sendErr)

			wantAttrs := attribute.NewSet(
				attribute.String("exporter", exporterID.String()),
				attribute.String(string(semconv.ErrorTypeKey), tt.expectedType),
				attribute.Bool(FailurePermanentKey, tt.isPermanent),
				attribute.Bool(FailureRetriesExhaustedKey, false),
			)

			metadatatest.AssertEqualExporterSendFailedSpans(t, telemetry,
				[]metricdata.DataPoint[int64]{
					{
						Attributes: wantAttrs,
						Value:      int64(req.Items),
					},
				}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
		})
	}
}

func TestExportTraceDataOp(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	parentCtx, parentSpan := tt.NewTelemetrySettings().TracerProvider.Tracer("test").Start(context.Background(), t.Name())
	defer parentSpan.End()

	var exporterErr error
	obsrep, err := newObsReportSender(
		exporter.Settings{ID: exporterID, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		pipeline.SignalTraces,
		sender.NewSender(func(context.Context, request.Request) error { return exporterErr }),
	)
	require.NoError(t, err)

	params := []testParams{
		{items: 22, err: nil},
		{items: 14, err: errFake},
	}
	for i := range params {
		exporterErr = params[i].err
		require.ErrorIs(t, obsrep.Send(parentCtx, &requesttest.FakeRequest{Items: params[i].items}), params[i].err)
	}

	spans := tt.SpanRecorder.Ended()
	require.Len(t, spans, len(params))

	var sentSpans, failedToSendSpans int
	for i, span := range spans {
		assert.Equal(t, "exporter/"+exporterID.String()+"/traces", span.Name())
		switch {
		case params[i].err == nil:
			sentSpans += params[i].items
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsSent, Value: attribute.Int64Value(int64(params[i].items))})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsFailed, Value: attribute.Int64Value(0)})
			assert.Equal(t, codes.Unset, span.Status().Code)
		case errors.Is(params[i].err, errFake):
			failedToSendSpans += params[i].items
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsSent, Value: attribute.Int64Value(0)})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsFailed, Value: attribute.Int64Value(int64(params[i].items))})
			assert.Equal(t, codes.Error, span.Status().Code)
			assert.Equal(t, params[i].err.Error(), span.Status().Description)
		default:
			t.Fatalf("unexpected error: %v", params[i].err)
		}
	}

	metadatatest.AssertEqualExporterSentSpans(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String("exporter", exporterID.String())),
				Value: int64(sentSpans),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())

	var expectedDataPoints []metricdata.DataPoint[int64]
	if failedToSendSpans > 0 {
		wantAttrs := attribute.NewSet(
			attribute.String("exporter", exporterID.String()),
			attribute.String(string(semconv.ErrorTypeKey), "Unknown"),
			attribute.Bool(FailurePermanentKey, false),
			attribute.Bool(FailureRetriesExhaustedKey, false),
		)
		expectedDataPoints = []metricdata.DataPoint[int64]{
			{
				Attributes: wantAttrs,
				Value:      int64(failedToSendSpans),
			},
		}
	}
	metadatatest.AssertEqualExporterSendFailedSpans(t, tt, expectedDataPoints,
		metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
}

func TestExportMetricsOp(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	parentCtx, parentSpan := tt.NewTelemetrySettings().TracerProvider.Tracer("test").Start(context.Background(), t.Name())
	defer parentSpan.End()

	var exporterErr error
	obsrep, err := newObsReportSender(
		exporter.Settings{ID: exporterID, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		pipeline.SignalMetrics,
		sender.NewSender(func(context.Context, request.Request) error { return exporterErr }),
	)
	require.NoError(t, err)

	params := []testParams{
		{items: 17, err: nil},
		{items: 23, err: errFake},
	}
	for i := range params {
		exporterErr = params[i].err
		require.ErrorIs(t, obsrep.Send(parentCtx, &requesttest.FakeRequest{Items: params[i].items}), params[i].err)
	}

	spans := tt.SpanRecorder.Ended()
	require.Len(t, spans, len(params))

	var sentMetricPoints, failedToSendMetricPoints int
	for i, span := range spans {
		assert.Equal(t, "exporter/"+exporterID.String()+"/metrics", span.Name())
		switch {
		case params[i].err == nil:
			sentMetricPoints += params[i].items
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsSent, Value: attribute.Int64Value(int64(params[i].items))})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsFailed, Value: attribute.Int64Value(0)})
			assert.Equal(t, codes.Unset, span.Status().Code)
		case errors.Is(params[i].err, errFake):
			failedToSendMetricPoints += params[i].items
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsSent, Value: attribute.Int64Value(0)})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsFailed, Value: attribute.Int64Value(int64(params[i].items))})
			assert.Equal(t, codes.Error, span.Status().Code)
			assert.Equal(t, params[i].err.Error(), span.Status().Description)
		default:
			t.Fatalf("unexpected error: %v", params[i].err)
		}
	}

	metadatatest.AssertEqualExporterSentMetricPoints(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String("exporter", exporterID.String())),
				Value: int64(sentMetricPoints),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())

	var expectedDataPoints []metricdata.DataPoint[int64]
	if failedToSendMetricPoints > 0 {
		wantAttrs := attribute.NewSet(
			attribute.String("exporter", exporterID.String()),
			attribute.String(string(semconv.ErrorTypeKey), "Unknown"),
			attribute.Bool(FailurePermanentKey, false),
			attribute.Bool(FailureRetriesExhaustedKey, false),
		)
		expectedDataPoints = []metricdata.DataPoint[int64]{
			{
				Attributes: wantAttrs,
				Value:      int64(failedToSendMetricPoints),
			},
		}
	}
	metadatatest.AssertEqualExporterSendFailedMetricPoints(t, tt, expectedDataPoints,
		metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
}

func TestExportLogsOp(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	parentCtx, parentSpan := tt.NewTelemetrySettings().TracerProvider.Tracer("test").Start(context.Background(), t.Name())
	defer parentSpan.End()

	var exporterErr error
	obsrep, err := newObsReportSender(
		exporter.Settings{ID: exporterID, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		pipeline.SignalLogs,
		sender.NewSender(func(context.Context, request.Request) error { return exporterErr }),
	)
	require.NoError(t, err)

	params := []testParams{
		{items: 17, err: nil},
		{items: 23, err: errFake},
	}
	for i := range params {
		exporterErr = params[i].err
		require.ErrorIs(t, obsrep.Send(parentCtx, &requesttest.FakeRequest{Items: params[i].items}), params[i].err)
	}

	spans := tt.SpanRecorder.Ended()
	require.Len(t, spans, len(params))

	var sentLogRecords, failedToSendLogRecords int
	for i, span := range spans {
		assert.Equal(t, "exporter/"+exporterID.String()+"/logs", span.Name())
		switch {
		case params[i].err == nil:
			sentLogRecords += params[i].items
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsSent, Value: attribute.Int64Value(int64(params[i].items))})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsFailed, Value: attribute.Int64Value(0)})
			assert.Equal(t, codes.Unset, span.Status().Code)
		case errors.Is(params[i].err, errFake):
			failedToSendLogRecords += params[i].items
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsSent, Value: attribute.Int64Value(0)})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsFailed, Value: attribute.Int64Value(int64(params[i].items))})
			assert.Equal(t, codes.Error, span.Status().Code)
			assert.Equal(t, params[i].err.Error(), span.Status().Description)
		default:
			t.Fatalf("unexpected error: %v", params[i].err)
		}
	}

	metadatatest.AssertEqualExporterSentLogRecords(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String("exporter", exporterID.String())),
				Value: int64(sentLogRecords),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())

	var expectedDataPoints []metricdata.DataPoint[int64]
	if failedToSendLogRecords > 0 {
		wantAttrs := attribute.NewSet(
			attribute.String("exporter", exporterID.String()),
			attribute.String(string(semconv.ErrorTypeKey), "Unknown"),
			attribute.Bool(FailurePermanentKey, false),
			attribute.Bool(FailureRetriesExhaustedKey, false),
		)
		expectedDataPoints = []metricdata.DataPoint[int64]{
			{
				Attributes: wantAttrs,
				Value:      int64(failedToSendLogRecords),
			},
		}
	}
	metadatatest.AssertEqualExporterSendFailedLogRecords(t, tt, expectedDataPoints,
		metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
}

// TestDetermineErrorType tests the determineErrorType function directly
func TestDetermineErrorType(t *testing.T) {
	tests := []struct {
		name              string
		err               error
		expectedErrorType string
	}{
		{
			name:              "nil error",
			err:               nil,
			expectedErrorType: "",
		},
		{
			name:              "retry exhausted error",
			err:               NewRetryExhaustedErr(errors.New("underlying error")),
			expectedErrorType: "Unknown",
		},
		{
			name:              "retry exhausted with gRPC error",
			err:               NewRetryExhaustedErr(status.Error(grpccodes.Unavailable, "service unavailable")),
			expectedErrorType: "Unavailable",
		},
		{
			name:              "shutdown error",
			err:               experr.NewShutdownErr(errors.New("shutting down")),
			expectedErrorType: "Shutdown",
		},
		{
			name:              "context canceled",
			err:               context.Canceled,
			expectedErrorType: "Canceled",
		},
		{
			name:              "context deadline exceeded",
			err:               context.DeadlineExceeded,
			expectedErrorType: "DeadlineExceeded",
		},
		{
			name:              "unknown error",
			err:               errors.New("some error"),
			expectedErrorType: "Unknown",
		},
		{
			name:              "wrapped context canceled",
			err:               fmt.Errorf("failed: %w", context.Canceled),
			expectedErrorType: "Canceled",
		},
		{
			name:              "wrapped context deadline exceeded",
			err:               fmt.Errorf("timeout: %w", context.DeadlineExceeded),
			expectedErrorType: "DeadlineExceeded",
		},
		{
			name:              "gRPC Unavailable",
			err:               status.Error(grpccodes.Unavailable, "service unavailable"),
			expectedErrorType: "Unavailable",
		},
		{
			name:              "gRPC ResourceExhausted",
			err:               status.Error(grpccodes.ResourceExhausted, "quota exceeded"),
			expectedErrorType: "ResourceExhausted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errorType := determineErrorType(tt.err)
			assert.Equal(t, tt.expectedErrorType, errorType, "error.type mismatch")
		})
	}
}

// TestExtractFailureAttributes tests the extractFailureAttributes function directly
func TestExtractFailureAttributes(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected attribute.Set
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: attribute.NewSet(),
		},
		{
			name: "retry exhausted error",
			err:  NewRetryExhaustedErr(errors.New("underlying error")),
			expected: attribute.NewSet(
				attribute.String(string(semconv.ErrorTypeKey), "Unknown"),
				attribute.Bool(FailurePermanentKey, false),
				attribute.Bool(FailureRetriesExhaustedKey, true),
			),
		},
		{
			name: "permanent error",
			err:  consumererror.NewPermanent(errors.New("bad data")),
			expected: attribute.NewSet(
				attribute.String(string(semconv.ErrorTypeKey), "Unknown"),
				attribute.Bool(FailurePermanentKey, true),
				attribute.Bool(FailureRetriesExhaustedKey, false),
			),
		},
		{
			name: "non-permanent error",
			err:  errors.New("transient error"),
			expected: attribute.NewSet(
				attribute.String(string(semconv.ErrorTypeKey), "Unknown"),
				attribute.Bool(FailurePermanentKey, false),
				attribute.Bool(FailureRetriesExhaustedKey, false),
			),
		},
		{
			name: "shutdown error",
			err:  experr.NewShutdownErr(errors.New("shutdown")),
			expected: attribute.NewSet(
				attribute.String(string(semconv.ErrorTypeKey), "Shutdown"),
				attribute.Bool(FailurePermanentKey, false),
				attribute.Bool(FailureRetriesExhaustedKey, false),
			),
		},
		{
			name: "context canceled",
			err:  context.Canceled,
			expected: attribute.NewSet(
				attribute.String(string(semconv.ErrorTypeKey), "Canceled"),
				attribute.Bool(FailurePermanentKey, false),
				attribute.Bool(FailureRetriesExhaustedKey, false),
			),
		},
		{
			name: "context deadline exceeded",
			err:  context.DeadlineExceeded,
			expected: attribute.NewSet(
				attribute.String(string(semconv.ErrorTypeKey), "DeadlineExceeded"),
				attribute.Bool(FailurePermanentKey, false),
				attribute.Bool(FailureRetriesExhaustedKey, false),
			),
		},
		{
			name: "gRPC Unavailable",
			err:  status.Error(grpccodes.Unavailable, "service unavailable"),
			expected: attribute.NewSet(
				attribute.String(string(semconv.ErrorTypeKey), "Unavailable"),
				attribute.Bool(FailurePermanentKey, false),
				attribute.Bool(FailureRetriesExhaustedKey, false),
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractFailureAttributes(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

type testParams struct {
	items int
	err   error
}
