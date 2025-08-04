// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package processorhelper // import "go.opentelemetry.io/collector/processor/processorhelper"

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/processor"
)

// ProcessMetricsFunc is a helper function that processes the incoming data and returns the data to be sent to the next component.
// If error is returned then returned data are ignored. It MUST not call the next component.
type ProcessMetricsFunc func(context.Context, pmetric.Metrics) (pmetric.Metrics, error)

type metrics struct {
	component.StartFunc
	component.ShutdownFunc
	consumer.Metrics
}

// NewMetrics creates a processor.Metrics that ensure context propagation and the right tags are set.
func NewMetrics(
	_ context.Context,
	set processor.Settings,
	_ component.Config,
	nextConsumer consumer.Metrics,
	metricsFunc ProcessMetricsFunc,
	options ...Option,
) (processor.Metrics, error) {
	if metricsFunc == nil {
		return nil, errors.New("nil metricsFunc")
	}

	obs, err := newObsReport(set, pipeline.SignalMetrics)
	if err != nil {
		return nil, err
	}

	eventOptions := spanAttributes(set.ID)
	bs := fromOptions(options)
	metricsConsumer, err := consumer.NewMetrics(func(ctx context.Context, md pmetric.Metrics) error {
		span := trace.SpanFromContext(ctx)
		span.AddEvent("Start processing.", eventOptions)

		startTime := time.Now()

		pointsIn := md.DataPointCount()

		var errFunc error
		md, errFunc = metricsFunc(ctx, md)
		obs.recordInternalDuration(ctx, startTime)
		span.AddEvent("End processing.", eventOptions)
		if errFunc != nil {
			obs.recordInOut(ctx, pointsIn, 0)
			if errors.Is(errFunc, ErrSkipProcessingData) {
				return nil
			}
			return errFunc
		}
		pointsOut := md.DataPointCount()
		obs.recordInOut(ctx, pointsIn, pointsOut)
		return nextConsumer.ConsumeMetrics(ctx, md)
	}, bs.consumerOptions...)
	if err != nil {
		return nil, err
	}

	return &metrics{
		StartFunc:    bs.StartFunc,
		ShutdownFunc: bs.ShutdownFunc,
		Metrics:      metricsConsumer,
	}, nil
}
