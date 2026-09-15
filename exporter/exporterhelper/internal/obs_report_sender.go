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

	// ErrorPermanentKey indicates whether the error is permanent (non-retryable).
	ErrorPermanentKey = "error.permanent"
)

type obsReportSender[K request.Request] struct {
	component.StartFunc
	component.ShutdownFunc

	spanName   string
	tracer     trace.Tracer
	spanAttrs  trace.SpanStartEventOption
	obsMetrics *ObsMetrics
	batch      bool
	next       sender.Sender[K]
}

func newObsReportSender[K request.Request](
	set exporter.Settings,
	signal pipeline.Signal,
	extraAttrs []attribute.KeyValue,
	batch bool,
	next sender.Sender[K],
) (sender.Sender[K], error) {
	return newObsReportSenderWithMetrics(set, signal, extraAttrs, nil, batch, next)
}

func newObsReportSenderWithMetrics[K request.Request](
	set exporter.Settings,
	signal pipeline.Signal,
	extraAttrs []attribute.KeyValue,
	obsMetrics *ObsMetrics,
	batch bool,
	next sender.Sender[K],
) (sender.Sender[K], error) {
	if obsMetrics == nil {
		var err error
		obsMetrics, err = newExporterObsReportMetrics(set, signal, extraAttrs)
		if err != nil {
			return nil, err
		}
	}
	idStr := set.ID.String()
	expAttr := attribute.String(ExporterKey, idStr)

	return &obsReportSender[K]{
		spanName:   ExporterKey + spanNameSep + idStr + spanNameSep + signal.String(),
		tracer:     metadata.Tracer(set.TelemetrySettings),
		spanAttrs:  trace.WithAttributes(expAttr, attribute.String(DataTypeKey, signal.String())),
		obsMetrics: obsMetrics,
		batch:      batch,
		next:       next,
	}, nil
}

func newExporterObsReportMetrics(
	set exporter.Settings,
	signal pipeline.Signal,
	extraAttrs []attribute.KeyValue,
) (*ObsMetrics, error) {
	tb, err := metadata.NewTelemetryBuilder(set.TelemetrySettings)
	if err != nil {
		return nil, err
	}

	exporterAttr := attribute.String(ExporterKey, set.ID.String())
	destinationAttrs := make([]attribute.KeyValue, 0, len(extraAttrs)+1)
	destinationAttrs = append(destinationAttrs, extraAttrs...)
	destinationAttrs = append(destinationAttrs, exporterAttr)
	metricAttr := metric.WithAttributeSet(attribute.NewSet(destinationAttrs...))
	inFlightAttr := metric.WithAttributeSet(attribute.NewSet(
		exporterAttr,
		attribute.String(DataTypeKey, signal.String()),
	))

	var itemsSentInst, itemsFailedInst metric.Int64Counter
	switch signal {
	case pipeline.SignalTraces:
		itemsSentInst = tb.ExporterSentSpans
		itemsFailedInst = tb.ExporterSendFailedSpans
	case pipeline.SignalMetrics:
		itemsSentInst = tb.ExporterSentMetricPoints
		itemsFailedInst = tb.ExporterSendFailedMetricPoints
	case pipeline.SignalLogs:
		itemsSentInst = tb.ExporterSentLogRecords
		itemsFailedInst = tb.ExporterSendFailedLogRecords
	case xpipeline.SignalProfiles:
		itemsSentInst = tb.ExporterSentProfileSamples
		itemsFailedInst = tb.ExporterSendFailedProfileSamples
	}

	return &ObsMetrics{
		BatchSendSizeFunc: func(ctx context.Context, items int64, bytesSize func() int64) {
			tb.ExporterQueueBatchSendSize.Record(ctx, items, metricAttr)
			if tb.ExporterQueueBatchSendSizeBytes.Enabled(ctx) {
				tb.ExporterQueueBatchSendSizeBytes.Record(ctx, bytesSize(), metricAttr)
			}
		},
		InFlightFunc: func(ctx context.Context, delta int64) {
			tb.ExporterInFlightRequests.Add(ctx, delta, inFlightAttr)
		},
		SentFunc: func(ctx context.Context, items int64) {
			if itemsSentInst != nil {
				itemsSentInst.Add(ctx, items, metricAttr)
			}
		},
		SendFailureFunc: func(ctx context.Context, items int64, options ...metric.AddOption) {
			if itemsFailedInst != nil {
				itemsFailedInst.Add(ctx, items, append([]metric.AddOption{metricAttr}, options...)...)
			}
		},
	}, nil
}

func (ors *obsReportSender[K]) Send(ctx context.Context, req K) error {
	// Have to read the number of items before sending the request since the request can
	// be modified by the downstream components like the batcher.
	c := ors.startOp(ctx)
	items := req.ItemsCount()
	if ors.batch {
		ors.obsMetrics.BatchSendSize(c, int64(items), func() int64 {
			return int64(req.BytesSize())
		})
	}
	// Forward the data to the next consumer (this pusher is the next).
	err := ors.next.Send(c, req)
	ors.endOp(c, items, err)
	return err
}

// startOp increments the in-flight request counter and creates the span
// used to trace the operation. Returns the updated context.
func (ors *obsReportSender[K]) startOp(ctx context.Context) context.Context {
	ors.obsMetrics.InFlight(ctx, 1)

	ctx, _ = ors.tracer.Start(ctx,
		ors.spanName,
		ors.spanAttrs,
		trace.WithLinks(queuebatch.LinksFromContext(ctx)...))
	return ctx
}

// EndOp completes the export operation that was started with StartOp.
func (ors *obsReportSender[K]) endOp(ctx context.Context, numRecords int, err error) {
	ors.obsMetrics.InFlight(ctx, -1)

	numSent, numFailedToSend := toNumItems(numRecords, err)

	ors.obsMetrics.Sent(ctx, numSent)

	if numFailedToSend > 0 {
		ors.obsMetrics.SendFailure(ctx, numFailedToSend, metric.WithAttributeSet(extractFailureAttributes(err)))
	}

	span := trace.SpanFromContext(ctx)
	defer span.End()
	// End the span according to errors.
	if span.IsRecording() {
		span.SetAttributes(
			attribute.Int64(ItemsSent, numSent),
			attribute.Int64(ItemsFailed, numFailedToSend),
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
