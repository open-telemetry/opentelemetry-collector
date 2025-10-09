// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:generate mdatagen metadata.yaml

package receiverhelper // import "go.opentelemetry.io/collector/receiver/receiverhelper"

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receiverhelper/internal"
	"go.opentelemetry.io/collector/receiver/receiverhelper/internal/metadata"
)

// ObsReport is a helper to add observability to a receiver.
type ObsReport struct {
	spanNamePrefix string
	transport      string
	longLivedCtx   bool
	tracer         trace.Tracer

	otelAttrs        metric.MeasurementOption
	telemetryBuilder *metadata.TelemetryBuilder
}

// ObsReportSettings are settings for creating an ObsReport.
type ObsReportSettings struct {
	ReceiverID component.ID
	Transport  string
	// LongLivedCtx when true indicates that the context passed in the call
	// outlives the individual receive operation.
	// Typically the long lived context is associated to a connection,
	// eg.: a gRPC stream, for which many batches of data are received in individual
	// operations without a corresponding new context per operation.
	LongLivedCtx           bool
	ReceiverCreateSettings receiver.Settings

	// prevent unkeyed literal initialization
	_ struct{}
}

// NewObsReport creates a new ObsReport.
func NewObsReport(cfg ObsReportSettings) (*ObsReport, error) {
	return newReceiver(cfg)
}

func newReceiver(cfg ObsReportSettings) (*ObsReport, error) {
	telemetryBuilder, err := metadata.NewTelemetryBuilder(cfg.ReceiverCreateSettings.TelemetrySettings)
	if err != nil {
		return nil, err
	}
	return &ObsReport{
		spanNamePrefix: internal.ReceiverKey + internal.SpanNameSep + cfg.ReceiverID.String(),
		transport:      cfg.Transport,
		longLivedCtx:   cfg.LongLivedCtx,
		tracer:         cfg.ReceiverCreateSettings.TracerProvider.Tracer(cfg.ReceiverID.String()),

		otelAttrs: metric.WithAttributeSet(attribute.NewSet(
			attribute.String(internal.ReceiverKey, cfg.ReceiverID.String()),
			attribute.String(internal.TransportKey, cfg.Transport),
		)),
		telemetryBuilder: telemetryBuilder,
	}, nil
}

// StartTracesOp is called when a request is received from a client.
// The returned context should be used in other calls to the obsreport functions
// dealing with the same receive operation.
func (rec *ObsReport) StartTracesOp(operationCtx context.Context) context.Context {
	return rec.startOp(operationCtx, internal.ReceiveTraceDataOperationSuffix)
}

// EndTracesOp completes the receive operation that was started with
// StartTracesOp.
func (rec *ObsReport) EndTracesOp(
	receiverCtx context.Context,
	format string,
	numReceivedSpans int,
	err error,
) {
	rec.endOp(receiverCtx, format, numReceivedSpans, err, pipeline.SignalTraces)
}

// StartLogsOp is called when a request is received from a client.
// The returned context should be used in other calls to the obsreport functions
// dealing with the same receive operation.
func (rec *ObsReport) StartLogsOp(operationCtx context.Context) context.Context {
	return rec.startOp(operationCtx, internal.ReceiverLogsOperationSuffix)
}

// EndLogsOp completes the receive operation that was started with
// StartLogsOp.
func (rec *ObsReport) EndLogsOp(
	receiverCtx context.Context,
	format string,
	numReceivedLogRecords int,
	err error,
) {
	rec.endOp(receiverCtx, format, numReceivedLogRecords, err, pipeline.SignalLogs)
}

// StartMetricsOp is called when a request is received from a client.
// The returned context should be used in other calls to the obsreport functions
// dealing with the same receive operation.
func (rec *ObsReport) StartMetricsOp(operationCtx context.Context) context.Context {
	return rec.startOp(operationCtx, internal.ReceiverMetricsOperationSuffix)
}

// EndMetricsOp completes the receive operation that was started with
// StartMetricsOp.
func (rec *ObsReport) EndMetricsOp(
	receiverCtx context.Context,
	format string,
	numReceivedPoints int,
	err error,
) {
	rec.endOp(receiverCtx, format, numReceivedPoints, err, pipeline.SignalMetrics)
}

// startOp creates the span used to trace the operation. Returning
// the updated context with the created span.
func (rec *ObsReport) startOp(receiverCtx context.Context, operationSuffix string) context.Context {
	var ctx context.Context
	var span trace.Span
	spanName := rec.spanNamePrefix + operationSuffix
	if !rec.longLivedCtx {
		ctx, span = rec.tracer.Start(receiverCtx, spanName)
	} else {
		// Since the receiverCtx is long lived do not use it to start the span.
		// This way this trace ends when the EndTracesOp is called.
		// Here is safe to ignore the returned context since it is not used below.
		_, span = rec.tracer.Start(context.Background(), spanName, trace.WithLinks(trace.Link{
			SpanContext: trace.SpanContextFromContext(receiverCtx),
		}))

		ctx = trace.ContextWithSpan(receiverCtx, span)
	}

	if rec.transport != "" {
		span.SetAttributes(attribute.String(internal.TransportKey, rec.transport))
	}
	return ctx
}

// endOp records the observability signals at the end of an operation.
func (rec *ObsReport) endOp(
	receiverCtx context.Context,
	format string,
	numReceivedItems int,
	err error,
	signal pipeline.Signal,
) {
	numAccepted := numReceivedItems
	numRefused := 0
	numFailedErrors := 0
	if err != nil {
		numAccepted = 0
		// If gate is enabled, we distinguish between refused and failed.
		if NewReceiverMetricsGate.IsEnabled() {
			if consumererror.IsDownstream(err) {
				numRefused = numReceivedItems
			} else {
				numFailedErrors = numReceivedItems
			}
		} else {
			// When the gate is disabled, all errors are considered "refused".
			numRefused = numReceivedItems
		}
	}

	span := trace.SpanFromContext(receiverCtx)

	rec.recordMetrics(receiverCtx, signal, numAccepted, numRefused, numFailedErrors)

	// The new otelcol_receiver_requests metric is only emitted when the feature gate is enabled.
	if NewReceiverMetricsGate.IsEnabled() {
		var outcome string
		switch {
		case err == nil:
			outcome = "success"
		case consumererror.IsDownstream(err):
			outcome = "refused"
		default:
			outcome = "failure"
		}
		rec.telemetryBuilder.ReceiverRequests.Add(receiverCtx, 1, rec.otelAttrs, metric.WithAttributeSet(attribute.NewSet(attribute.String("outcome", outcome))))
	}

	// end span according to errors
	if span.IsRecording() {
		var acceptedItemsKey, refusedItemsKey, failedItemsKey string
		switch signal {
		case pipeline.SignalTraces:
			acceptedItemsKey = internal.AcceptedSpansKey
			refusedItemsKey = internal.RefusedSpansKey
			failedItemsKey = internal.FailedSpansKey
		case pipeline.SignalMetrics:
			acceptedItemsKey = internal.AcceptedMetricPointsKey
			refusedItemsKey = internal.RefusedMetricPointsKey
			failedItemsKey = internal.FailedMetricPointsKey
		case pipeline.SignalLogs:
			acceptedItemsKey = internal.AcceptedLogRecordsKey
			refusedItemsKey = internal.RefusedLogRecordsKey
			failedItemsKey = internal.FailedLogRecordsKey
		}

		span.SetAttributes(
			attribute.String(internal.FormatKey, format),
			attribute.Int64(acceptedItemsKey, int64(numAccepted)),
			attribute.Int64(refusedItemsKey, int64(numRefused)),
			attribute.Int64(failedItemsKey, int64(numFailedErrors)),
		)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
		}
	}
	span.End()
}

func (rec *ObsReport) recordMetrics(receiverCtx context.Context, signal pipeline.Signal, numAccepted, numRefused, numFailedErrors int) {
	var acceptedMeasure, refusedMeasure, failedMeasure metric.Int64Counter
	switch signal {
	case pipeline.SignalTraces:
		acceptedMeasure = rec.telemetryBuilder.ReceiverAcceptedSpans
		refusedMeasure = rec.telemetryBuilder.ReceiverRefusedSpans
		failedMeasure = rec.telemetryBuilder.ReceiverFailedSpans
	case pipeline.SignalMetrics:
		acceptedMeasure = rec.telemetryBuilder.ReceiverAcceptedMetricPoints
		refusedMeasure = rec.telemetryBuilder.ReceiverRefusedMetricPoints
		failedMeasure = rec.telemetryBuilder.ReceiverFailedMetricPoints
	case pipeline.SignalLogs:
		acceptedMeasure = rec.telemetryBuilder.ReceiverAcceptedLogRecords
		refusedMeasure = rec.telemetryBuilder.ReceiverRefusedLogRecords
		failedMeasure = rec.telemetryBuilder.ReceiverFailedLogRecords
	}

	acceptedMeasure.Add(receiverCtx, int64(numAccepted), rec.otelAttrs)
	refusedMeasure.Add(receiverCtx, int64(numRefused), rec.otelAttrs)
	failedMeasure.Add(receiverCtx, int64(numFailedErrors), rec.otelAttrs)
}
