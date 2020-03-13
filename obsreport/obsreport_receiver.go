// Copyright 2020 OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package obsreport

import (
	"context"

	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.opencensus.io/trace"

	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/observability"
)

const (
	ReceiverKey  = "receiver"
	TransportKey = "transport"
	FormatKey    = "format"

	AcceptedSpansKey = "accepted_spans"
	RefusedSpansKey  = "refused_spans"

	AcceptedMetricPointsKey = "accepted_metric_points"
	RefusedMetricPointsKey  = "refused_metric_points"
)

var (
	tagKeyReceiver, _  = tag.NewKey(ReceiverKey)
	tagKeyTransport, _ = tag.NewKey(TransportKey)

	receiverPrefix                  = ReceiverKey + nameSep
	receiveTraceDataOperationSuffix = nameSep + "TraceDataReceived"
	receiverMetricsOperationSuffix  = nameSep + "MetricsReceived"

	// Receiver metrics. Any count of data items below is in the original format
	// that they were received, reasoning: reconciliation is easier if measurements
	// on clients and receiver are expected to be the same. Translation issues
	// that result in a different number of elements should be reported in a
	// separate way.
	mReceiverAcceptedSpans = stats.Int64(
		receiverPrefix+AcceptedSpansKey,
		"Number of spans successfully pushed into the pipeline.",
		stats.UnitDimensionless)
	mReceiverRefusedSpans = stats.Int64(
		receiverPrefix+RefusedSpansKey,
		"Number of spans that could not be pushed into the pipeline.",
		stats.UnitDimensionless)
	mReceiverAcceptedMetricPoints = stats.Int64(
		receiverPrefix+AcceptedMetricPointsKey,
		"Number of metric points successfully pushed into the pipeline.",
		stats.UnitDimensionless)
	mReceiverRefusedMetricPoints = stats.Int64(
		receiverPrefix+RefusedMetricPointsKey,
		"Number of metric points that could not be pushed into the pipeline.",
		stats.UnitDimensionless)
)

// StartReceiveOptions has the options related to starting a receive operation.
type StartReceiveOptions struct {
	// LongLivedCtx is used in cases when individual receive messages happen
	// under a single long lived context.
	LongLivedCtx context.Context

	// AlwaysSample is used typically in tests to enforce the span added to the
	// operation context to be sampled.
	AlwaysSample bool
}

// StartReceiveOption function applues changes to StartReceiveOptions.
type StartReceiveOption func(*StartReceiveOptions)

// WithLongLivedCtx starts a single receive operation under the long lived
// context of the RPC or connection.
func WithLongLivedCtx(longLivedCtx context.Context) StartReceiveOption {
	return func(opts *StartReceiveOptions) {
		opts.LongLivedCtx = longLivedCtx
	}
}

// WithAlwaysSample is used to force the span generated for the operation to be
// sampled, typically it is used for tests.
func WithAlwaysSample() StartReceiveOption {
	return func(opts *StartReceiveOptions) {
		opts.AlwaysSample = true
	}
}

// StartTraceDataReceiveOp is called when a \request is received from a client.
// The returned context should be used in other calls to the obsreport functions
// dealing with the same receive operation.
func StartTraceDataReceiveOp(
	operationCtx context.Context,
	receiver string,
	transport string,
	opt ...StartReceiveOption,
) context.Context {
	return traceReceiveOp(
		operationCtx,
		receiver,
		transport,
		receiveTraceDataOperationSuffix,
		opt...)
}

// EndTraceDataReceiveOp completes the receive operation that was started with
// StartTraceDataReceiveOp.
func EndTraceDataReceiveOp(
	receiverCtx context.Context,
	format string,
	numReceivedSpans int,
	err error,
) {
	if useLegacy {
		numReceivedLegacy := numReceivedSpans
		numDroppedSpans := 0
		if err != nil {
			numDroppedSpans = numReceivedSpans
			numReceivedLegacy = 0
		}
		observability.RecordMetricsForTraceReceiver(
			receiverCtx, numReceivedLegacy, numDroppedSpans)
	}

	endReceiveOp(
		receiverCtx,
		format,
		numReceivedSpans,
		err,
		configmodels.TracesDataType,
	)
}

// StartMetricsReceiveOp is called when a \request is received from a client.
// The returned context should be used in other calls to the obsreport functions
// dealing with the same receive operation.
func StartMetricsReceiveOp(
	operationCtx context.Context,
	receiver string,
	transport string,
	opt ...StartReceiveOption,
) context.Context {
	return traceReceiveOp(
		operationCtx,
		receiver,
		transport,
		receiverMetricsOperationSuffix,
		opt...)
}

// EndMetricsReceiveOp completes the receive operation that was started with
// StartMetricsReceiveOp.
func EndMetricsReceiveOp(
	receiverCtx context.Context,
	format string,
	numReceivedPoints int,
	numReceivedTimeSeries int, // For legacy measurements.
	err error,
) {
	if useLegacy {
		numDroppedTimeSeries := 0
		if err != nil {
			numDroppedTimeSeries = numReceivedTimeSeries
			numReceivedTimeSeries = 0
		}
		observability.RecordMetricsForMetricsReceiver(
			receiverCtx, numReceivedTimeSeries, numDroppedTimeSeries)
	}

	endReceiveOp(
		receiverCtx,
		format,
		numReceivedPoints,
		err,
		configmodels.MetricsDataType,
	)
}

// ReceiverContext adds the keys used when recording observability metrics to
// the given context returning the newly created context. This context should
// be used in related calls to the obsreport functions so metrics are properly
// recorded.
func ReceiverContext(
	ctx context.Context,
	receiver string,
	transport string,
	legacyName string,
) context.Context {
	if useLegacy {
		name := receiver
		if legacyName != "" {
			name = legacyName
		}
		ctx = observability.ContextWithReceiverName(ctx, name)
	}

	if useNew {
		mutators := []tag.Mutator{
			tag.Upsert(tagKeyReceiver, receiver, tag.WithTTL(tag.TTLNoPropagation)),
			tag.Upsert(tagKeyTransport, transport, tag.WithTTL(tag.TTLNoPropagation)),
		}
		ctx, _ = tag.New(ctx, mutators...)
	}

	return ctx
}

// traceReceiveOp creates the span used to trace the operation. Returning
// the updated context with the created span.
func traceReceiveOp(
	receiverCtx context.Context,
	receiverName string,
	transport string,
	operationSuffix string,
	opt ...StartReceiveOption,
) context.Context {
	var opts StartReceiveOptions
	for _, o := range opt {
		o(&opts)
	}

	var sampleOption []trace.StartOption
	if opts.AlwaysSample {
		sampleOption = []trace.StartOption{
			trace.WithSampler(trace.AlwaysSample()),
		}
	}

	var ctx context.Context
	var span *trace.Span
	spanName := receiverPrefix + receiverName + operationSuffix
	if opts.LongLivedCtx == nil {
		ctx, span = trace.StartSpan(receiverCtx, spanName, sampleOption...)
	} else {
		// Do not use LongLivedRPCCtx to start the span so this trace ends when
		// the EndTraceDataReceiveOp is called. Here is safe to ignore the returned
		// context since it is not used below.
		_, span = trace.StartSpan(receiverCtx, spanName, sampleOption...)

		// If the long lived context has a parent span, then add it as a parent link.
		setParentLink(opts.LongLivedCtx, span)

		ctx = trace.NewContext(opts.LongLivedCtx, span)
	}

	span.AddAttributes(trace.StringAttribute(
		TransportKey, transport))
	return ctx
}

// endReceiveOp records the observability signals at the end of an operation.
func endReceiveOp(
	receiverCtx context.Context,
	format string,
	numReceivedItems int,
	err error,
	dataType configmodels.DataType,
) {
	numAccepted := numReceivedItems
	numRefused := 0
	if err != nil {
		numAccepted = 0
		numRefused = numReceivedItems
	}

	span := trace.FromContext(receiverCtx)

	if useNew {
		var acceptedMeasure, refusedMeasure *stats.Int64Measure
		switch dataType {
		case configmodels.TracesDataType:
			acceptedMeasure = mReceiverAcceptedSpans
			refusedMeasure = mReceiverRefusedSpans
		case configmodels.MetricsDataType:
			acceptedMeasure = mReceiverAcceptedMetricPoints
			refusedMeasure = mReceiverRefusedMetricPoints
		}

		stats.Record(
			receiverCtx,
			acceptedMeasure.M(int64(numAccepted)),
			refusedMeasure.M(int64(numRefused)))
	}

	// end span according to errors
	if span.IsRecordingEvents() {
		var acceptedItemsKey, refusedItemsKey string
		switch dataType {
		case configmodels.TracesDataType:
			acceptedItemsKey = AcceptedSpansKey
			refusedItemsKey = RefusedSpansKey
		case configmodels.MetricsDataType:
			acceptedItemsKey = AcceptedMetricPointsKey
			refusedItemsKey = RefusedMetricPointsKey
		}

		span.AddAttributes(
			trace.StringAttribute(
				FormatKey, format),
			trace.Int64Attribute(
				acceptedItemsKey, int64(numAccepted)),
			trace.Int64Attribute(
				refusedItemsKey, int64(numRefused)),
		)
		span.SetStatus(errToStatus(err))
	}
	span.End()
}
