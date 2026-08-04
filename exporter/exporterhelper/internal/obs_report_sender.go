// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/experr"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/metadata"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
)

const (
	// spanNameSep is duplicate between receiver and exporter.
	spanNameSep = "/"

	// ExporterKey used to identify exporters in metrics and traces.
	ExporterKey = "exporter"

	// DataTypeKey used to identify the data type in the queue size metric.
	DataTypeKey = "data_type"

	// ItemsSent used to track number of items sent by exporters.
	ItemsSent = "items.sent"
	// ItemsFailed used to track number of items that failed to be sent by exporters.
	ItemsFailed = "items.failed"
	// ItemsDropped used to track number of items intentionally dropped by exporters.
	ItemsDropped = "items.dropped"

	// ErrorPermanentKey indicates whether the error is permanent (non-retryable).
	ErrorPermanentKey = "error.permanent"
	// DroppedReasonKey carries the optional reason from a DroppedItemsErr on the
	// exporter_dropped_* metrics. The attribute is only retained at the detailed
	// telemetry level (filtered out by a metric view at lower levels) because
	// the reason string is exporter-defined and may have unbounded cardinality.
	DroppedReasonKey = "exporter.dropped.reason"
)

type obsReportSender[K request.Request] struct {
	component.StartFunc
	component.ShutdownFunc

	logger             *zap.Logger
	exporterIDStr      string
	spanName           string
	tracer             trace.Tracer
	spanAttrs          trace.SpanStartEventOption
	metricAttr         metric.MeasurementOption
	inFlightMetricAttr metric.MeasurementOption
	itemsSentInst      metric.Int64Counter
	itemsFailedInst    metric.Int64Counter
	itemsDroppedInst   metric.Int64Counter
	inFlightInst       metric.Int64UpDownCounter
	next               sender.Sender[K]
}

func newObsReportSender[K request.Request](set exporter.Settings, signal pipeline.Signal, extraAttrs []attribute.KeyValue, next sender.Sender[K]) (sender.Sender[K], error) {
	telemetryBuilder, err := metadata.NewTelemetryBuilder(set.TelemetrySettings)
	if err != nil {
		return nil, err
	}

	idStr := set.ID.String()
	expAttr := attribute.String(ExporterKey, idStr)

	or := &obsReportSender[K]{
		logger:             set.Logger,
		exporterIDStr:      idStr,
		spanName:           ExporterKey + spanNameSep + idStr + spanNameSep + signal.String(),
		tracer:             metadata.Tracer(set.TelemetrySettings),
		spanAttrs:          trace.WithAttributes(expAttr, attribute.String(DataTypeKey, signal.String())),
		metricAttr:         metric.WithAttributeSet(attribute.NewSet(append(extraAttrs, expAttr)...)),
		inFlightMetricAttr: metric.WithAttributeSet(attribute.NewSet(expAttr, attribute.String(DataTypeKey, signal.String()))),
		next:               next,
	}

	or.inFlightInst = telemetryBuilder.ExporterInFlightRequests

	switch signal {
	case pipeline.SignalTraces:
		or.itemsSentInst = telemetryBuilder.ExporterSentSpans
		or.itemsFailedInst = telemetryBuilder.ExporterSendFailedSpans
		or.itemsDroppedInst = telemetryBuilder.ExporterDroppedSpans

	case pipeline.SignalMetrics:
		or.itemsSentInst = telemetryBuilder.ExporterSentMetricPoints
		or.itemsFailedInst = telemetryBuilder.ExporterSendFailedMetricPoints
		or.itemsDroppedInst = telemetryBuilder.ExporterDroppedMetricPoints

	case pipeline.SignalLogs:
		or.itemsSentInst = telemetryBuilder.ExporterSentLogRecords
		or.itemsFailedInst = telemetryBuilder.ExporterSendFailedLogRecords
		or.itemsDroppedInst = telemetryBuilder.ExporterDroppedLogRecords

	case xpipeline.SignalProfiles:
		or.itemsSentInst = telemetryBuilder.ExporterSentProfileSamples
		or.itemsFailedInst = telemetryBuilder.ExporterSendFailedProfileSamples
		or.itemsDroppedInst = telemetryBuilder.ExporterDroppedProfileSamples
	}

	return or, nil
}

func (ors *obsReportSender[K]) Send(ctx context.Context, req K) error {
	// Have to read the number of items before sending the request since the request can
	// be modified by the downstream components like the batcher.
	c := ors.startOp(ctx)
	items := req.ItemsCount()
	// Forward the data to the next consumer (this pusher is the next).
	err := ors.next.Send(c, req)
	ors.endOp(c, items, err)
	// DroppedItemsErr is a non-failure sentinel: don't propagate it as an
	// error to the rest of the pipeline.
	if _, ok := experr.DroppedItemsFromErr(err); ok {
		return nil
	}
	return err
}

// startOp increments the in-flight request counter and creates the span
// used to trace the operation. Returns the updated context.
func (ors *obsReportSender[K]) startOp(ctx context.Context) context.Context {
	if ors.inFlightInst != nil {
		ors.inFlightInst.Add(ctx, 1, ors.inFlightMetricAttr)
	}

	ctx, _ = ors.tracer.Start(ctx,
		ors.spanName,
		ors.spanAttrs,
		trace.WithLinks(queuebatch.LinksFromContext(ctx)...))
	return ctx
}

// EndOp completes the export operation that was started with StartOp.
func (ors *obsReportSender[K]) endOp(ctx context.Context, numRecords int, err error) {
	if ors.inFlightInst != nil {
		ors.inFlightInst.Add(ctx, -1, ors.inFlightMetricAttr)
	}

	// Check if the exporter intentionally dropped some items.  A
	// DroppedItemsErr is a non-failure sentinel: the exporter processed the
	// request successfully but chose to discard certain items (e.g. a
	// Prometheus exporter dropping non-monotonic DELTA sums).
	var (
		numDropped    int64
		droppedReason string
	)
	if d, ok := experr.DroppedItemsFromErr(err); ok {
		numDropped = int64(d.Dropped)
		droppedReason = d.Reason
		// Clear the error so that it is not counted as a send failure and
		// is not propagated to the rest of the pipeline.
		err = nil
	}

	numSent, numFailedToSend := toNumItems(numRecords, err)

	// Items that were intentionally dropped should not be counted as
	// successfully sent.  Clamp to zero in case the exporter reported more
	// dropped items than the total item count, and warn — that condition
	// indicates an exporter bug.
	if numDropped > 0 && numSent <= numDropped {
		if ors.logger != nil {
			ors.logger.Warn("exporter reported more dropped items than the request contained",
				zap.String("exporter", ors.exporterIDStr),
				zap.Int64("dropped", numDropped),
				zap.Int64("items", numSent),
				zap.String("reason", droppedReason),
			)
		}
		numSent = 0
	} else if numDropped > 0 {
		numSent -= numDropped
	}

	if ors.itemsSentInst != nil {
		ors.itemsSentInst.Add(ctx, numSent, ors.metricAttr)
	}

	if ors.itemsFailedInst != nil && numFailedToSend > 0 {
		withFailedAttrs := metric.WithAttributeSet(extractFailureAttributes(err))
		ors.itemsFailedInst.Add(ctx, numFailedToSend, ors.metricAttr, withFailedAttrs)
	}

	if ors.itemsDroppedInst != nil && numDropped > 0 {
		withReason := metric.WithAttributeSet(
			attribute.NewSet(attribute.String(DroppedReasonKey, droppedReason)),
		)
		ors.itemsDroppedInst.Add(ctx, numDropped, ors.metricAttr, withReason)
	}

	span := trace.SpanFromContext(ctx)
	defer span.End()
	// End the span according to errors.
	if span.IsRecording() {
		span.SetAttributes(
			attribute.Int64(ItemsSent, numSent),
			attribute.Int64(ItemsFailed, numFailedToSend),
			attribute.Int64(ItemsDropped, numDropped),
		)
		if err != nil {
			span.SetStatus(otelcodes.Error, err.Error())
		}
	}
}

func toNumItems(numExportedItems int, err error) (int64, int64) {
	if err != nil {
		return 0, int64(numExportedItems)
	}
	return int64(numExportedItems), 0
}

func extractFailureAttributes(err error) attribute.Set {
	if err == nil {
		return attribute.NewSet()
	}

	attrs := []attribute.KeyValue{}

	errorType := determineErrorType(err)
	attrs = append(attrs, attribute.String(string(semconv.ErrorTypeKey), errorType))

	isPermanent := consumererror.IsPermanent(err)
	attrs = append(attrs, attribute.Bool(ErrorPermanentKey, isPermanent))

	return attribute.NewSet(attrs...)
}

func determineErrorType(err error) string {
	if experr.IsShutdownErr(err) {
		return "Shutdown"
	}

	if errors.Is(err, context.Canceled) {
		return "Canceled"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "Deadline_Exceeded"
	}

	if st, ok := status.FromError(err); ok && st.Code() != codes.OK {
		return st.Code().String()
	}

	return "_OTHER"
}
