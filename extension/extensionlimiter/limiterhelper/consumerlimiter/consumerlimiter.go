// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumerlimiter // import "go.opentelemetry.io/collector/extension/extensionlimiter/limiterhelper/consumerlimiter"

import (
	"context"
	"slices"

	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/extension/extensionlimiter"
	"go.opentelemetry.io/collector/extension/extensionlimiter/limiterhelper"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/xreceiver"
)

// Config is the standard pipeline configuration for limiting a
// consumer interface by specific signal.
type Config struct {
	RequestCount component.ID `mapstructure:"request_count"`
	RequestItems component.ID `mapstructure:"request_items"`
	RequestBytes component.ID `mapstructure:"request_bytes"`
}

// Traits object interface is generalized by P the pipeline data type
// (e.g., ptrace.Traces) and C the consumer type (e.g.,
// consumer.Traces)
type traits[P, C any] interface {
	// itemCount is SpanCount(), DataPointCount(), or LogRecordCount().
	itemCount(P) int
	// requestBytes uses the appropriate protobuf Bytesr as a proxy
	// for memory used.
	requestSize(data P) int
	// consume calls the appropriate consumer method (e.g., ConsumeTraces)
	consume(ctx context.Context, data P, next C) error
	// create is a functional constructor the consumer type (e.g., consumer.NewTraces)
	create(func(ctx context.Context, data P) error, ...consumer.Option) (C, error)
}

// Traces traits

type traceTraits struct{}

func (traceTraits) itemCount(data ptrace.Traces) int {
	return data.SpanCount()
}

func (traceTraits) requestSize(data ptrace.Traces) int {
	var sizer ptrace.MarshalSizer
	return sizer.TracesSize(data)
}

func (traceTraits) create(next func(ctx context.Context, data ptrace.Traces) error, opts ...consumer.Option) (consumer.Traces, error) {
	return consumer.NewTraces(next, opts...)
}

func (traceTraits) consume(ctx context.Context, data ptrace.Traces, next consumer.Traces) error {
	return next.ConsumeTraces(ctx, data)
}

// Metrics traits

type metricTraits struct{}

func (metricTraits) itemCount(data pmetric.Metrics) int {
	return data.DataPointCount()
}

func (metricTraits) requestSize(data pmetric.Metrics) int {
	var sizer pmetric.MarshalSizer
	return sizer.MetricsSize(data)
}

func (metricTraits) create(next func(ctx context.Context, data pmetric.Metrics) error, opts ...consumer.Option) (consumer.Metrics, error) {
	return consumer.NewMetrics(next, opts...)
}

func (metricTraits) consume(ctx context.Context, data pmetric.Metrics, next consumer.Metrics) error {
	return next.ConsumeMetrics(ctx, data)
}

// Logs traits

type logTraits struct{}

func (logTraits) itemCount(data plog.Logs) int {
	return data.LogRecordCount()
}

func (logTraits) requestSize(data plog.Logs) int {
	var sizer plog.MarshalSizer
	return sizer.LogsSize(data)
}

func (logTraits) create(next func(ctx context.Context, data plog.Logs) error, opts ...consumer.Option) (consumer.Logs, error) {
	return consumer.NewLogs(next, opts...)
}

func (logTraits) consume(ctx context.Context, data plog.Logs, next consumer.Logs) error {
	return next.ConsumeLogs(ctx, data)
}

// Profiles traits

type profileTraits struct{}

func (profileTraits) itemCount(data pprofile.Profiles) int {
	return data.SampleCount()
}

func (profileTraits) requestSize(data pprofile.Profiles) int {
	var sizer pprofile.MarshalSizer
	return sizer.ProfilesSize(data)
}

func (profileTraits) create(next func(ctx context.Context, data pprofile.Profiles) error, opts ...consumer.Option) (xconsumer.Profiles, error) {
	return xconsumer.NewProfiles(next, opts...)
}

func (profileTraits) consume(ctx context.Context, data pprofile.Profiles, next xconsumer.Profiles) error {
	return next.ConsumeProfiles(ctx, data)
}

// limitOne obtains a Wrapper and applies a single weight limit.
func limitOne[P any, C any](
	next C,
	keys []extensionlimiter.WeightKey,
	provider limiterhelper.WrapperProvider,
	m traits[P, C],
	key extensionlimiter.WeightKey,
	opts []consumer.Option,
	quantify func(P) int,
) (C, error) {
	if !slices.Contains(keys, key) {
		return next, nil
	}
	lim, err := provider.GetWrapper(key)
	if err != nil {
		return next, err
	}
	if lim == nil {
		return next, nil
	}
	return m.create(func(ctx context.Context, data P) error {
		return lim.LimitCall(ctx, quantify(data), func(ctx context.Context) error {
			return m.consume(ctx, data, next)
		})
	}, opts...)
}

// newLimited is signal-generic limiting logic.
func newLimited[P any, C any](
	next C,
	keys []extensionlimiter.WeightKey,
	provider limiterhelper.WrapperProvider,
	m traits[P, C],
	opts ...consumer.Option,
) (C, error) {
	if provider == nil {
		return next, nil
	}
	var err1, err2, err3 error
	// Note: reverse order of evaluation cost => least-cost applied first.
	next, err1 = limitOne(next, keys, provider, m, extensionlimiter.WeightKeyRequestBytes, opts,
		func(data P) int {
			return m.requestSize(data)
		})
	next, err2 = limitOne(next, keys, provider, m, extensionlimiter.WeightKeyRequestItems, opts,
		func(data P) int {
			return m.itemCount(data)
		})
	next, err3 = limitOne(next, keys, provider, m, extensionlimiter.WeightKeyRequestCount, opts,
		func(_ P) int {
			return 1
		})
	return next, multierr.Append(err1, multierr.Append(err2, err3))
}

// // NewLimitedTraces applies a limiter using the provider over keys before calling next.
// func NewLimitedTraces(next consumer.Traces, keys []extensionlimiter.WeightKey, provider limiterhelper.WrapperProvider) (consumer.Traces, error) {
// 	return newLimited(next, keys, provider, traceTraits{},
// 		consumer.WithCapabilities(next.Capabilities()))
// }

// // NewLimitedLogs applies a limiter using the provider over keys before calling next.
// func NewLimitedLogs(next consumer.Logs, keys []extensionlimiter.WeightKey, provider limiterhelper.WrapperProvider) (consumer.Logs, error) {
// 	return newLimited(next, keys, provider, logTraits{},
// 		consumer.WithCapabilities(next.Capabilities()))
// }

// // NewLimitedMetrics applies a limiter using the provider over keys before calling next.
// func NewLimitedMetrics(next consumer.Metrics, keys []extensionlimiter.WeightKey, provider limiterhelper.WrapperProvider) (consumer.Metrics, error) {
// 	return newLimited(next, keys, provider, metricTraits{},
// 		consumer.WithCapabilities(next.Capabilities()))
// }

// // NewLimitedProfiles applies a limiter using the provider over keys before calling next.
// func NewLimitedProfiles(next xconsumer.Profiles, keys []extensionlimiter.WeightKey, provider limiterhelper.WrapperProvider) (xconsumer.Profiles, error) {
// 	return newLimited(next, keys, provider, profileTraits{},
// 		consumer.WithCapabilities(next.Capabilities()))
// }

// type stabilityFunc func() component.StabilityLevel

// 	// TracesStability gets the stability level of the Traces receiver.
// 	TracesStability() component.StabilityLevel
// 	// MetricsStability gets the stability level of the Metrics receiver.
// 	MetricsStability() component.StabilityLevel

func NewLimitedFactory(fact xreceiver.Factory) xreceiver.Factory {
	return xreceiver.NewFactoryImpl(
		receiver.NewFactoryImpl(
			component.NewFactoryImpl(
				fact.Type,
				fact.CreateDefaultConfig,
			),
			fact.CreateTraces,
			fact.TracesStability,
			fact.CreateMetrics,
			fact.MetricsStability,
			fact.CreateLogs,
			fact.LogsStability,
		),
		fact.CreateProfiles,
		fact.ProfilesStability,
	)
}
