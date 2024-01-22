// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receiverhelper // import "go.opentelemetry.io/collector/receiver/receiverhelper"

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
	"go.opentelemetry.io/collector/receiver"
)

const (
	receiverScope = obsmetrics.Scope + obsmetrics.NameSep + obsmetrics.ReceiverKey
)

// ObsReport is a helper to add observability to a receiver.
type ObsReport struct {
	level          configtelemetry.Level
	spanNamePrefix string
	transport      string
	longLivedCtx   bool
	tracer         trace.Tracer
	meter          metric.Meter
	logger         *zap.Logger

	otelAttrs []attribute.KeyValue

	acceptedSpansCounter        metric.Int64Counter
	refusedSpansCounter         metric.Int64Counter
	acceptedMetricPointsCounter metric.Int64Counter
	refusedMetricPointsCounter  metric.Int64Counter
	acceptedLogRecordsCounter   metric.Int64Counter
	refusedLogRecordsCounter    metric.Int64Counter
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
	ReceiverCreateSettings receiver.CreateSettings
}

// NewObsReport creates a new ObsReport.
func NewObsReport(cfg ObsReportSettings) (*ObsReport, error) {
	return newReceiver(cfg)
}

func newReceiver(cfg ObsReportSettings) (*ObsReport, error) {
	rec := &ObsReport{
		level:          cfg.ReceiverCreateSettings.TelemetrySettings.MetricsLevel,
		spanNamePrefix: obsmetrics.ReceiverPrefix + cfg.ReceiverID.String(),
		transport:      cfg.Transport,
		longLivedCtx:   cfg.LongLivedCtx,
		tracer:         cfg.ReceiverCreateSettings.TracerProvider.Tracer(cfg.ReceiverID.String()),
		meter:          cfg.ReceiverCreateSettings.MeterProvider.Meter(receiverScope),
		logger:         cfg.ReceiverCreateSettings.Logger,

		otelAttrs: []attribute.KeyValue{
			attribute.String(obsmetrics.ReceiverKey, cfg.ReceiverID.String()),
			attribute.String(obsmetrics.TransportKey, cfg.Transport),
		},
	}

	if err := rec.createOtelMetrics(); err != nil {
		return nil, err
	}

	return rec, nil
}

func (rec *ObsReport) createOtelMetrics() error {
	var errors, err error

	rec.acceptedSpansCounter, err = rec.meter.Int64Counter(
		obsmetrics.ReceiverPrefix+obsmetrics.AcceptedSpansKey,
		metric.WithDescription("Number of spans successfully pushed into the pipeline."),
		metric.WithUnit("1"),
	)
	errors = multierr.Append(errors, err)

	rec.refusedSpansCounter, err = rec.meter.Int64Counter(
		obsmetrics.ReceiverPrefix+obsmetrics.RefusedSpansKey,
		metric.WithDescription("Number of spans that could not be pushed into the pipeline."),
		metric.WithUnit("1"),
	)
	errors = multierr.Append(errors, err)

	rec.acceptedMetricPointsCounter, err = rec.meter.Int64Counter(
		obsmetrics.ReceiverPrefix+obsmetrics.AcceptedMetricPointsKey,
		metric.WithDescription("Number of metric points successfully pushed into the pipeline."),
		metric.WithUnit("1"),
	)
	errors = multierr.Append(errors, err)

	rec.refusedMetricPointsCounter, err = rec.meter.Int64Counter(
		obsmetrics.ReceiverPrefix+obsmetrics.RefusedMetricPointsKey,
		metric.WithDescription("Number of metric points that could not be pushed into the pipeline."),
		metric.WithUnit("1"),
	)
	errors = multierr.Append(errors, err)

	rec.acceptedLogRecordsCounter, err = rec.meter.Int64Counter(
		obsmetrics.ReceiverPrefix+obsmetrics.AcceptedLogRecordsKey,
		metric.WithDescription("Number of log records successfully pushed into the pipeline."),
		metric.WithUnit("1"),
	)
	errors = multierr.Append(errors, err)

	rec.refusedLogRecordsCounter, err = rec.meter.Int64Counter(
		obsmetrics.ReceiverPrefix+obsmetrics.RefusedLogRecordsKey,
		metric.WithDescription("Number of log records that could not be pushed into the pipeline."),
		metric.WithUnit("1"),
	)
	errors = multierr.Append(errors, err)

	return errors
}

// StartTracesOp is called when a request is received from a client.
// The returned context should be used in other calls to the obsreport functions
// dealing with the same receive operation.
func (rec *ObsReport) StartTracesOp(operationCtx context.Context) context.Context {
	return rec.startOp(operationCtx, obsmetrics.ReceiveTraceDataOperationSuffix)
}

// EndTracesOp completes the receive operation that was started with
// StartTracesOp.
func (rec *ObsReport) EndTracesOp(
	receiverCtx context.Context,
	format string,
	numReceivedSpans int,
	err error,
) {
	rec.endOp(receiverCtx, format, numReceivedSpans, err, component.DataTypeTraces)
}

// StartLogsOp is called when a request is received from a client.
// The returned context should be used in other calls to the obsreport functions
// dealing with the same receive operation.
func (rec *ObsReport) StartLogsOp(operationCtx context.Context) context.Context {
	return rec.startOp(operationCtx, obsmetrics.ReceiverLogsOperationSuffix)
}

// EndLogsOp completes the receive operation that was started with
// StartLogsOp.
func (rec *ObsReport) EndLogsOp(
	receiverCtx context.Context,
	format string,
	numReceivedLogRecords int,
	err error,
) {
	rec.endOp(receiverCtx, format, numReceivedLogRecords, err, component.DataTypeLogs)
}

// StartMetricsOp is called when a request is received from a client.
// The returned context should be used in other calls to the obsreport functions
// dealing with the same receive operation.
func (rec *ObsReport) StartMetricsOp(operationCtx context.Context) context.Context {
	return rec.startOp(operationCtx, obsmetrics.ReceiverMetricsOperationSuffix)
}

// EndMetricsOp completes the receive operation that was started with
// StartMetricsOp.
func (rec *ObsReport) EndMetricsOp(
	receiverCtx context.Context,
	format string,
	numReceivedPoints int,
	err error,
) {
	rec.endOp(receiverCtx, format, numReceivedPoints, err, component.DataTypeMetrics)
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
		span.SetAttributes(attribute.String(obsmetrics.TransportKey, rec.transport))
	}
	return ctx
}

// endOp records the observability signals at the end of an operation.
func (rec *ObsReport) endOp(
	receiverCtx context.Context,
	format string,
	numReceivedItems int,
	err error,
	dataType component.DataType,
) {
	numAccepted := numReceivedItems
	numRefused := 0
	if err != nil {
		numAccepted = 0
		numRefused = numReceivedItems
	}

	span := trace.SpanFromContext(receiverCtx)

	if rec.level != configtelemetry.LevelNone {
		rec.recordMetrics(receiverCtx, dataType, numAccepted, numRefused)
	}

	// end span according to errors
	if span.IsRecording() {
		var acceptedItemsKey, refusedItemsKey string
		switch dataType {
		case component.DataTypeTraces:
			acceptedItemsKey = obsmetrics.AcceptedSpansKey
			refusedItemsKey = obsmetrics.RefusedSpansKey
		case component.DataTypeMetrics:
			acceptedItemsKey = obsmetrics.AcceptedMetricPointsKey
			refusedItemsKey = obsmetrics.RefusedMetricPointsKey
		case component.DataTypeLogs:
			acceptedItemsKey = obsmetrics.AcceptedLogRecordsKey
			refusedItemsKey = obsmetrics.RefusedLogRecordsKey
		}

		span.SetAttributes(
			attribute.String(obsmetrics.FormatKey, format),
			attribute.Int64(acceptedItemsKey, int64(numAccepted)),
			attribute.Int64(refusedItemsKey, int64(numRefused)),
		)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
		}
	}
	span.End()
}

func (rec *ObsReport) recordMetrics(receiverCtx context.Context, dataType component.DataType, numAccepted, numRefused int) {
	var acceptedMeasure, refusedMeasure metric.Int64Counter
	switch dataType {
	case component.DataTypeTraces:
		acceptedMeasure = rec.acceptedSpansCounter
		refusedMeasure = rec.refusedSpansCounter
	case component.DataTypeMetrics:
		acceptedMeasure = rec.acceptedMetricPointsCounter
		refusedMeasure = rec.refusedMetricPointsCounter
	case component.DataTypeLogs:
		acceptedMeasure = rec.acceptedLogRecordsCounter
		refusedMeasure = rec.refusedLogRecordsCounter
	}

	acceptedMeasure.Add(receiverCtx, int64(numAccepted), metric.WithAttributes(rec.otelAttrs...))
	refusedMeasure.Add(receiverCtx, int64(numRefused), metric.WithAttributes(rec.otelAttrs...))
}
