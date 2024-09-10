// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package processorhelper // import "go.opentelemetry.io/collector/processor/processorhelper"

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"
)

// ProcessMetricsFunc is a helper function that processes the incoming data and returns the data to be sent to the next component.
// If error is returned then returned data are ignored. It MUST not call the next component.
type ProcessMetricsFunc func(context.Context, pmetric.Metrics) (pmetric.Metrics, error)

type metricsProcessor struct {
	component.StartFunc
	component.ShutdownFunc
	consumer.Metrics
}

// NewMetricsProcessor creates a processor.Metrics that ensure context propagation and the right tags are set.
func NewMetricsProcessor(
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

	obs, err := newObsReport(ObsReportSettings{
		ProcessorID:             set.ID,
		ProcessorCreateSettings: set,
	})
	if err != nil {
		return nil, err
	}
	obs.otelAttrs = append(obs.otelAttrs, attribute.String("otel.signal", "metrics"))

	eventOptions := spanAttributes(set.ID)
	bs := fromOptions(options)
	marshaler := new(pmetric.ProtoMarshaler)
	metricsConsumer, err := consumer.NewMetrics(func(ctx context.Context, md pmetric.Metrics) error {
		var pointsIn, pointsOut, bytesIn, bytesOut int

		span := trace.SpanFromContext(ctx)
		span.AddEvent("Start processing.", eventOptions)
		pointsIn = md.DataPointCount()
		if obs.metricLevel >= configtelemetry.LevelDetailed {
			bytesIn = marshaler.MetricsSize(md)
		}
		md, err = metricsFunc(ctx, md)
		span.AddEvent("End processing.", eventOptions)
		if err != nil {
			if errors.Is(err, ErrSkipProcessingData) {
				return nil
			}
			return err
		}
		pointsOut = md.DataPointCount()
		obs.recordInOut(ctx, pointsIn, pointsOut)
		if obs.metricLevel >= configtelemetry.LevelDetailed {
			bytesOut = marshaler.MetricsSize(md)
			obs.recordInOutSize(ctx, bytesIn, bytesOut)
		}
		return nextConsumer.ConsumeMetrics(ctx, md)
	}, bs.consumerOptions...)
	if err != nil {
		return nil, err
	}

	return &metricsProcessor{
		StartFunc:    bs.StartFunc,
		ShutdownFunc: bs.ShutdownFunc,
		Metrics:      metricsConsumer,
	}, nil
}
