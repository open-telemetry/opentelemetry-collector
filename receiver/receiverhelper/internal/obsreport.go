// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/receiver/receiverhelper/internal"

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
	"go.opentelemetry.io/collector/receiver/receiverhelper/internal/metadata"
)

// SplitItems splits numReceivedItems into accepted, refused and failed items
// depending on err.
func SplitItems(numReceivedItems int, err error) (numAccepted, numRefused, numFailedErrors int) {
	if err == nil {
		return numReceivedItems, 0, 0
	}
	// If gate is enabled, we distinguish between refused and failed.
	if metadata.ReceiverhelperNewReceiverMetricsFeatureGate.IsEnabled() && !consumererror.IsDownstream(err) {
		return 0, 0, numReceivedItems
	}
	// When the gate is disabled, all errors are considered "refused".
	return 0, numReceivedItems, 0
}

// RecordRequests records the outcome of a receive operation. The new
// otelcol_receiver_requests metric is only emitted when the feature gate is
// enabled.
func RecordRequests(ctx context.Context, requests metric.Int64Counter, otelAttrs metric.MeasurementOption, err error) {
	if !metadata.ReceiverhelperNewReceiverMetricsFeatureGate.IsEnabled() {
		return
	}
	var outcome string
	switch {
	case err == nil:
		outcome = "success"
	case consumererror.IsDownstream(err):
		outcome = "refused"
	default:
		outcome = "failure"
	}
	requests.Add(ctx, 1, otelAttrs, metric.WithAttributeSet(attribute.NewSet(attribute.String("outcome", outcome))))
}

// EndSpan records the item counts of a receive operation on span and ends it
// according to err.
func EndSpan(
	span trace.Span,
	format string,
	signal pipeline.Signal,
	numAccepted, numRefused, numFailedErrors int,
	err error,
) {
	if span.IsRecording() {
		var acceptedItemsKey, refusedItemsKey, failedItemsKey string
		switch signal {
		case pipeline.SignalTraces:
			acceptedItemsKey = AcceptedSpansKey
			refusedItemsKey = RefusedSpansKey
			failedItemsKey = FailedSpansKey
		case pipeline.SignalMetrics:
			acceptedItemsKey = AcceptedMetricPointsKey
			refusedItemsKey = RefusedMetricPointsKey
			failedItemsKey = FailedMetricPointsKey
		case pipeline.SignalLogs:
			acceptedItemsKey = AcceptedLogRecordsKey
			refusedItemsKey = RefusedLogRecordsKey
			failedItemsKey = FailedLogRecordsKey
		case xpipeline.SignalProfiles:
			acceptedItemsKey = AcceptedProfileSamplesKey
			refusedItemsKey = RefusedProfileSamplesKey
			failedItemsKey = FailedProfileSamplesKey
		}

		span.SetAttributes(
			attribute.String(FormatKey, format),
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
