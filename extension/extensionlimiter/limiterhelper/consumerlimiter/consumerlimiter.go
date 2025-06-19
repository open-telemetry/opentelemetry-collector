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
//
// This type should be embedded, for example:
//
// 	// LimiterConfig allows applying limiter extensions for request count, items, and bytes.
//	consumerlimiter.LimiterConfig `mapstructure:"limiters"`
//
type LimiterConfig struct {
	RequestCount component.ID `mapstructure:"request_count"`
	RequestItems component.ID `mapstructure:"request_items"`
	RequestBytes component.ID `mapstructure:"request_bytes"`
}

type HasLimiterConfig interface {
	getConfig() LimiterConfig
}

func (c LimiterConfig) getConfig() LimiterConfig {
	return c
}

var _ HasLimiterConfig = LimiterConfig{}

type capable interface {
	Capabilities() consumer.Capabilities
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

var _ traits[ptrace.Traces, consumer.Traces] = traceTraits{}

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

var _ traits[pmetric.Metrics, consumer.Metrics] = metricTraits{}

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

var _ traits[plog.Logs, consumer.Logs] = logTraits{}

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

var _ traits[pprofile.Profiles, xconsumer.Profiles] = profileTraits{}

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
func limitOne[P, C, R any](
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

type limitedReceiver[P any, C capable, T traits[P, C]] struct {
	next C

	component.ShutdownFunc
}

func (l *limitedReceiver[P, C, T]) Capabilities() consumer.Capabilities {
	return l.next.Capabilities()
}

func (l *limitedReceiver[P, C, T]) Start(ctx context.Context, host component.Host) error {
	// @@@
	return nil
}

func (l *limitedReceiver[P, C, T]) consume(ctx context.Context, data P) error {
	var t T
	return t.consume(ctx, data, l.next)
}

// newLimited is signal-generic limiting logic.
func newLimited[P any, C capable](
	next C,
	cfg LimiterConfig,
	t traits[P, C],
) *limitedReceiver[P, C, traits[P, C]] { 
	return &limitedReceiver[P, C, traits[P, C]]{
		next: next,
	}
}
	// opts ...consumer.Option,
	// if provider == nil {
	// 	return next, nil
	// }
	// var err1, err2, err3 error
	// // Note: reverse order of evaluation cost => least-cost applied first.
	// next, err1 = limitOne(next, keys, provider, m, extensionlimiter.WeightKeyRequestBytes, opts,
	// 	func(data P) int {
	// 		return m.requestSize(data)
	// 	})
	// next, err2 = limitOne(next, keys, provider, m, extensionlimiter.WeightKeyRequestItems, opts,
	// 	func(data P) int {
	// 		return m.itemCount(data)
	// 	})
	// next, err3 = limitOne(next, keys, provider, m, extensionlimiter.WeightKeyRequestCount, opts,
	// 	func(_ P) int {
	// 		return 1
	// 	})
	// return next, multierr.Append(err1, multierr.Append(err2, err3))
	//var zero C
	// @@@ @@@!!!
	//return  zero, nil

type creator[C capable, R component.Component] func(ctx context.Context, set receiver.Settings, cfg component.Config, next C) (R, error)

// limitReceiver limits a receiver component where P is pipeline data,
// C is the consumer type, and R is the return type.
func limitReceiver[P any, C capable, R component.Component](
	cf creator[C, R],
	t traits[P, C],
) creator[C, R] {
	return func(ctx context.Context, set receiver.Settings, cfg component.Config, next C) (R, error) {
		var limiter *limitedReceiver[P, C, traits[P, C]]
		if lc, ok := cfg.(HasLimiterConfig); ok {
			limiter = newLimited(next, lc.getConfig(), t)
			var err error
			next, err = t.create(limiter.consume)
			if err != nil {
				var zero R
				return zero, err
			}
		}

		recv, err := cf(ctx, set, cfg, next)
		if err != nil {
			return recv, err
		}
		return component.NewComponentImpl(
			func (ctx context.Context, host component.Host) error {
				err1 := limiter.Start(ctx, host)
				err2 := recv.Start(ctx, host)
				return multierr.Append(err1, err2) 
			},
			func (ctx context.Context) error {
				err1 := recv.Shutdown(ctx)
				err2 := limiter.Shutdown(ctx)
				return multierr.Append(err1, err2) 
			},
		), nil
	}
}

func NewLimitedFactory(fact xreceiver.Factory) xreceiver.Factory {
	return xreceiver.NewFactoryImpl(
		receiver.NewFactoryImpl(
			component.NewFactoryImpl(
				fact.Type,
				fact.CreateDefaultConfig,
			),
			receiver.CreateTracesFunc(limitReceiver(fact.CreateTraces, traceTraits{})),
			fact.TracesStability,
			receiver.CreateMetricsFunc(limitReceiver(fact.CreateMetrics, metricTraits{})),
			fact.MetricsStability,
			receiver.CreateLogsFunc(limitReceiver(fact.CreateLogs, logTraits{})),
			fact.LogsStability,
		),
		xreceiver.CreateProfilesFunc(limitReceiver(fact.CreateProfiles, profileTraits{})),
		fact.ProfilesStability,
	)
}
