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

package obsreport // import "go.opentelemetry.io/collector/obsreport"

import (
	"context"

	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/instrument/syncint64"
	"go.opentelemetry.io/otel/metric/unit"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/internal/obsreportconfig"
	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
)

const (
	receiverName = "receiver"

	receiverScope = scopeName + nameSep + receiverName
)

// Receiver is a helper to add observability to a component.Receiver.
type Receiver struct {
	level          configtelemetry.Level
	spanNamePrefix string
	transport      string
	longLivedCtx   bool
	mutators       []tag.Mutator
	tracer         trace.Tracer
	meter          metric.Meter
	logger         *zap.Logger

	useOtelForMetrics bool
	otelAttrs         []attribute.KeyValue

	acceptedSpansCounter        syncint64.Counter
	refusedSpansCounter         syncint64.Counter
	acceptedMetricPointsCounter syncint64.Counter
	refusedMetricPointsCounter  syncint64.Counter
	acceptedLogRecordsCounter   syncint64.Counter
	refusedLogRecordsCounter    syncint64.Counter
}

// ReceiverSettings are settings for creating an Receiver.
type ReceiverSettings struct {
	ReceiverID config.ComponentID
	Transport  string
	// LongLivedCtx when true indicates that the context passed in the call
	// outlives the individual receive operation.
	// Typically the long lived context is associated to a connection,
	// eg.: a gRPC stream, for which many batches of data are received in individual
	// operations without a corresponding new context per operation.
	LongLivedCtx           bool
	ReceiverCreateSettings component.ReceiverCreateSettings
}

// NewReceiver creates a new Receiver.
func NewReceiver(cfg ReceiverSettings) *Receiver {
	rec := &Receiver{
		level:          cfg.ReceiverCreateSettings.TelemetrySettings.MetricsLevel,
		spanNamePrefix: obsmetrics.ReceiverPrefix + cfg.ReceiverID.String(),
		transport:      cfg.Transport,
		longLivedCtx:   cfg.LongLivedCtx,
		mutators: []tag.Mutator{
			tag.Upsert(obsmetrics.TagKeyReceiver, cfg.ReceiverID.String(), tag.WithTTL(tag.TTLNoPropagation)),
			tag.Upsert(obsmetrics.TagKeyTransport, cfg.Transport, tag.WithTTL(tag.TTLNoPropagation)),
		},
		tracer: cfg.ReceiverCreateSettings.TracerProvider.Tracer(cfg.ReceiverID.String()),
		meter:  cfg.ReceiverCreateSettings.MeterProvider.Meter(receiverScope),
		logger: cfg.ReceiverCreateSettings.Logger,

		useOtelForMetrics: featuregate.GetRegistry().IsEnabled(obsreportconfig.UseOtelForInternalMetricsfeatureGateID),
		otelAttrs: []attribute.KeyValue{
			attribute.String(obsmetrics.ReceiverKey, cfg.ReceiverID.String()),
			attribute.String(obsmetrics.TransportKey, cfg.Transport),
		},
	}

	rec.createOtelMetrics()

	return rec
}

func (rec *Receiver) createOtelMetrics() {
	if !rec.useOtelForMetrics {
		return
	}

	var err error
	handleError := func(metricName string, err error) {
		if err != nil {
			rec.logger.Warn("failed to create otel instrument", zap.Error(err), zap.String("metric", metricName))
		}
	}

	rec.acceptedSpansCounter, err = rec.meter.SyncInt64().Counter(
		obsmetrics.ReceiverPrefix+obsmetrics.AcceptedSpansKey,
		instrument.WithDescription("Number of spans successfully pushed into the pipeline."),
		instrument.WithUnit(unit.Dimensionless),
	)
	handleError(obsmetrics.ReceiverPrefix+obsmetrics.AcceptedSpansKey, err)

	rec.refusedSpansCounter, err = rec.meter.SyncInt64().Counter(
		obsmetrics.ReceiverPrefix+obsmetrics.RefusedSpansKey,
		instrument.WithDescription("Number of spans that could not be pushed into the pipeline."),
		instrument.WithUnit(unit.Dimensionless),
	)
	handleError(obsmetrics.ReceiverPrefix+obsmetrics.RefusedSpansKey, err)

	rec.acceptedMetricPointsCounter, err = rec.meter.SyncInt64().Counter(
		obsmetrics.ReceiverPrefix+obsmetrics.AcceptedMetricPointsKey,
		instrument.WithDescription("Number of metric points successfully pushed into the pipeline."),
		instrument.WithUnit(unit.Dimensionless),
	)
	handleError(obsmetrics.ReceiverPrefix+obsmetrics.AcceptedMetricPointsKey, err)

	rec.refusedMetricPointsCounter, err = rec.meter.SyncInt64().Counter(
		obsmetrics.ReceiverPrefix+obsmetrics.RefusedMetricPointsKey,
		instrument.WithDescription("Number of metric points that could not be pushed into the pipeline."),
		instrument.WithUnit(unit.Dimensionless),
	)
	handleError(obsmetrics.ReceiverPrefix+obsmetrics.RefusedMetricPointsKey, err)

	rec.acceptedLogRecordsCounter, err = rec.meter.SyncInt64().Counter(
		obsmetrics.ReceiverPrefix+obsmetrics.AcceptedLogRecordsKey,
		instrument.WithDescription("Number of log records successfully pushed into the pipeline."),
		instrument.WithUnit(unit.Dimensionless),
	)
	handleError(obsmetrics.ReceiverPrefix+obsmetrics.AcceptedLogRecordsKey, err)

	rec.refusedLogRecordsCounter, err = rec.meter.SyncInt64().Counter(
		obsmetrics.ReceiverPrefix+obsmetrics.RefusedLogRecordsKey,
		instrument.WithDescription("Number of log records that could not be pushed into the pipeline."),
		instrument.WithUnit(unit.Dimensionless),
	)
	handleError(obsmetrics.ReceiverPrefix+obsmetrics.RefusedLogRecordsKey, err)
}

// StartTracesOp is called when a request is received from a client.
// The returned context should be used in other calls to the obsreport functions
// dealing with the same receive operation.
func (rec *Receiver) StartTracesOp(operationCtx context.Context) context.Context {
	return rec.startOp(operationCtx, obsmetrics.ReceiveTraceDataOperationSuffix)
}

// EndTracesOp completes the receive operation that was started with
// StartTracesOp.
func (rec *Receiver) EndTracesOp(
	receiverCtx context.Context,
	format string,
	numReceivedSpans int,
	err error,
) {
	rec.endOp(receiverCtx, format, numReceivedSpans, err, config.TracesDataType)
}

// StartLogsOp is called when a request is received from a client.
// The returned context should be used in other calls to the obsreport functions
// dealing with the same receive operation.
func (rec *Receiver) StartLogsOp(operationCtx context.Context) context.Context {
	return rec.startOp(operationCtx, obsmetrics.ReceiverLogsOperationSuffix)
}

// EndLogsOp completes the receive operation that was started with
// StartLogsOp.
func (rec *Receiver) EndLogsOp(
	receiverCtx context.Context,
	format string,
	numReceivedLogRecords int,
	err error,
) {
	rec.endOp(receiverCtx, format, numReceivedLogRecords, err, config.LogsDataType)
}

// StartMetricsOp is called when a request is received from a client.
// The returned context should be used in other calls to the obsreport functions
// dealing with the same receive operation.
func (rec *Receiver) StartMetricsOp(operationCtx context.Context) context.Context {
	return rec.startOp(operationCtx, obsmetrics.ReceiverMetricsOperationSuffix)
}

// EndMetricsOp completes the receive operation that was started with
// StartMetricsOp.
func (rec *Receiver) EndMetricsOp(
	receiverCtx context.Context,
	format string,
	numReceivedPoints int,
	err error,
) {
	rec.endOp(receiverCtx, format, numReceivedPoints, err, config.MetricsDataType)
}

// startOp creates the span used to trace the operation. Returning
// the updated context with the created span.
func (rec *Receiver) startOp(receiverCtx context.Context, operationSuffix string) context.Context {
	ctx, _ := tag.New(receiverCtx, rec.mutators...)
	var span trace.Span
	spanName := rec.spanNamePrefix + operationSuffix
	if !rec.longLivedCtx {
		ctx, span = rec.tracer.Start(ctx, spanName)
	} else {
		// Since the receiverCtx is long lived do not use it to start the span.
		// This way this trace ends when the EndTracesOp is called.
		// Here is safe to ignore the returned context since it is not used below.
		_, span = rec.tracer.Start(context.Background(), spanName, trace.WithLinks(trace.Link{
			SpanContext: trace.SpanContextFromContext(receiverCtx),
		}))

		ctx = trace.ContextWithSpan(ctx, span)
	}

	if rec.transport != "" {
		span.SetAttributes(attribute.String(obsmetrics.TransportKey, rec.transport))
	}
	return ctx
}

// endOp records the observability signals at the end of an operation.
func (rec *Receiver) endOp(
	receiverCtx context.Context,
	format string,
	numReceivedItems int,
	err error,
	dataType config.DataType,
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
		case config.TracesDataType:
			acceptedItemsKey = obsmetrics.AcceptedSpansKey
			refusedItemsKey = obsmetrics.RefusedSpansKey
		case config.MetricsDataType:
			acceptedItemsKey = obsmetrics.AcceptedMetricPointsKey
			refusedItemsKey = obsmetrics.RefusedMetricPointsKey
		case config.LogsDataType:
			acceptedItemsKey = obsmetrics.AcceptedLogRecordsKey
			refusedItemsKey = obsmetrics.RefusedLogRecordsKey
		}

		span.SetAttributes(
			attribute.String(obsmetrics.FormatKey, format),
			attribute.Int64(acceptedItemsKey, int64(numAccepted)),
			attribute.Int64(refusedItemsKey, int64(numRefused)),
		)
		recordError(span, err)
	}
	span.End()
}

func (rec *Receiver) recordMetrics(receiverCtx context.Context, dataType config.DataType, numAccepted, numRefused int) {
	if rec.useOtelForMetrics {
		rec.recordWithOtel(receiverCtx, dataType, numAccepted, numRefused)
	} else {
		rec.recordWithOC(receiverCtx, dataType, numAccepted, numRefused)
	}
}

func (rec *Receiver) recordWithOtel(receiverCtx context.Context, dataType config.DataType, numAccepted, numRefused int) {
	var acceptedMeasure, refusedMeasure syncint64.Counter
	switch dataType {
	case config.TracesDataType:
		acceptedMeasure = rec.acceptedSpansCounter
		refusedMeasure = rec.refusedSpansCounter
	case config.MetricsDataType:
		acceptedMeasure = rec.acceptedMetricPointsCounter
		refusedMeasure = rec.refusedMetricPointsCounter
	case config.LogsDataType:
		acceptedMeasure = rec.acceptedLogRecordsCounter
		refusedMeasure = rec.refusedLogRecordsCounter
	}

	acceptedMeasure.Add(receiverCtx, int64(numAccepted), rec.otelAttrs...)
	refusedMeasure.Add(receiverCtx, int64(numRefused), rec.otelAttrs...)
}

func (rec *Receiver) recordWithOC(receiverCtx context.Context, dataType config.DataType, numAccepted, numRefused int) {
	var acceptedMeasure, refusedMeasure *stats.Int64Measure
	switch dataType {
	case config.TracesDataType:
		acceptedMeasure = obsmetrics.ReceiverAcceptedSpans
		refusedMeasure = obsmetrics.ReceiverRefusedSpans
	case config.MetricsDataType:
		acceptedMeasure = obsmetrics.ReceiverAcceptedMetricPoints
		refusedMeasure = obsmetrics.ReceiverRefusedMetricPoints
	case config.LogsDataType:
		acceptedMeasure = obsmetrics.ReceiverAcceptedLogRecords
		refusedMeasure = obsmetrics.ReceiverRefusedLogRecords
	}

	stats.Record(
		receiverCtx,
		acceptedMeasure.M(int64(numAccepted)),
		refusedMeasure.M(int64(numRefused)))
}
