// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumerlimiter // import "go.opentelemetry.io/collector/extension/extensionlimiter/limiterhelper/consumerlimiter"

import (
	"context"
	"errors"
	"fmt"

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

var (
	ErrLimiterNotFound    = errors.New("limiter not found")
	ErrNotALimiter        = errors.New("not a limiter")
	ErrLimiterUnsupported = errors.New("limiter unsupported")
)

// LimiterConfig is the standard pipeline configuration for limiting a
// consumer interface by specific signal. This is identical to
// configmiddleware.Config.
type Config component.ID

// UnmarshalText implements encoding.TextUnmarshaler for YAML support.
func (cfg *Config) UnmarshalText(text []byte) error {
	var id component.ID
	err := id.UnmarshalText(text)
	*cfg = Config(id)
	return err
}

// capable is an internal interface describing common features of a
// consumer.
type capable interface {
	Capabilities() consumer.Capabilities
}

// Traits object interface is generalized by P the pipeline data type
// (e.g., ptrace.Traces) and C the consumer type (e.g.,
// consumer.Traces) and R the return component type.
type traits[P, C, R any] interface {
	// itemCount is SpanCount(), DataPointCount(), or LogRecordCount().
	itemCount(P) int
	// requestBytes uses the appropriate protobuf Bytesr as a proxy
	// for memory used.
	requestSize(data P) int
	// consume calls the appropriate consumer method (e.g., ConsumeTraces)
	consume(ctx context.Context, data P, next C) error
	// create is a functional constructor the consumer type (e.g., consumer.NewTraces)
	create(func(ctx context.Context, data P) error, ...consumer.Option) (C, error)
	// newReceiver constructs the correct receiver type
	newReceiver(component.Component) R
}

// Creator is the function to create a receiver components.
type creator[C capable, R component.Component] func(ctx context.Context, set receiver.Settings, cfg component.Config, next C) (R, error)

// Traces traits

type traceTraits struct{}

var _ traits[ptrace.Traces, consumer.Traces, receiver.Traces] = traceTraits{}

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

func (traceTraits) newReceiver(c component.Component) receiver.Traces {
	return receiver.Traces(c)
}

// Metrics traits

type metricTraits struct{}

var _ traits[pmetric.Metrics, consumer.Metrics, receiver.Metrics] = metricTraits{}

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

func (metricTraits) newReceiver(c component.Component) receiver.Metrics {
	return receiver.Metrics(c)
}

// Logs traits

type logTraits struct{}

var _ traits[plog.Logs, consumer.Logs, receiver.Logs] = logTraits{}

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

func (logTraits) newReceiver(c component.Component) receiver.Logs {
	return receiver.Logs(c)
}

// Profiles traits

type profileTraits struct{}

var _ traits[pprofile.Profiles, xconsumer.Profiles, xreceiver.Profiles] = profileTraits{}

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

func (profileTraits) newReceiver(c component.Component) xreceiver.Profiles {
	return xreceiver.Profiles(c)
}

type limitedReceiver[P any, C capable, R component.Component, T traits[P, C, R]] struct {
	cfg  Config
	next C
	self T
	component.ShutdownFunc
}

func (l *limitedReceiver[P, C, R, T]) Capabilities() consumer.Capabilities {
	return l.next.Capabilities()
}

func (l *limitedReceiver[P, C, R, T]) Start(ctx context.Context, host component.Host) error {
	var unset component.ID
	if component.ID(l.cfg) == unset {
		// No limiter configured
		return nil
	}

	limiterID := component.ID(l.cfg)
	var err1, err2, err3 error

	// Apply the same limiter to all three weight keys
	l.next, err1 = l.limitOne(
		host,
		limiterID,
		extensionlimiter.WeightKeyRequestBytes,
		func(data P) int {
			return l.self.requestSize(data)
		},
	)
	l.next, err2 = l.limitOne(
		host,
		limiterID,
		extensionlimiter.WeightKeyRequestItems,
		func(data P) int {
			return l.self.itemCount(data)
		},
	)
	l.next, err3 = l.limitOne(
		host,
		limiterID,
		extensionlimiter.WeightKeyRequestCount,
		func(data P) int {
			return 1
		},
	)
	
	return multierr.Append(err1, multierr.Append(err2, err3))
}

func (l *limitedReceiver[P, C, R, T]) consume(ctx context.Context, data P) error {
	return l.self.consume(ctx, data, l.next)
}

// limitOne obtains a Wrapper and applies a single weight limit.
func (l *limitedReceiver[P, C, R, T]) limitOne(
	host component.Host,
	name component.ID,
	key extensionlimiter.WeightKey,
	quantify func(P) int,
) (C, error) {
	exts := host.GetExtensions()
	comp := exts[name]
	if comp == nil {
		return l.next, fmt.Errorf("%w: %s", ErrLimiterNotFound, name.String())
	}
	alim, isLim := comp.(extensionlimiter.AnyProvider)
	if !isLim {
		return l.next, fmt.Errorf("%w: %s", ErrNotALimiter, name.String())
	}
	provider, err := limiterhelper.AnyToWrapperProvider(alim)
	if err != nil {
		return l.next, err
	}
	// Note: not passing options to GetWrapper(), an open question.
	lim, err := provider.GetWrapper(key)
	if err != nil {
		return l.next, err
	}
	if lim == nil {
		return l.next, nil
	}
	return l.self.create(func(ctx context.Context, data P) error {
		// Ensure limiter tracking exists in context and get tracking state
		ctx, tracking := extensionlimiter.EnsureLimiterTracking(ctx)
		amount := quantify(data)

		// Check if any limiter was already applied for this weight key
		if tracking.HasBeenLimited(key) {
			// Skip this limiter - already applied (likely in middleware)
			return l.self.consume(ctx, data, l.next)
		}

		// Apply limiter and increment the counter for this weight key
		return lim.LimitCall(ctx, amount, func(ctx context.Context) error {
			if tracking != nil {
				tracking.AddRequest(key, amount)
			}
			return l.self.consume(ctx, data, l.next)
		})
	}, consumer.WithCapabilities(l.next.Capabilities()))
}

// newLimited is signal-generic limiting logic.
func newLimited[P any, C capable, R component.Component](
	next C,
	cfg Config,
	self traits[P, C, R],
) *limitedReceiver[P, C, R, traits[P, C, R]] {
	return &limitedReceiver[P, C, R, traits[P, C, R]]{
		cfg:  cfg,
		next: next,
		self: self,
	}
}

// LimiterConfigurator lets components configure limiters using
// a field they determine.
type LimiterConfigurator func(component.Config) Config

// limitReceiver limits a receiver component where P is pipeline data,
// C is the consumer type, and R is the return type.
func limitReceiver[P any, C capable, R component.Component](
	cf creator[C, R],
	t traits[P, C, R],
	cfgf LimiterConfigurator,
) creator[C, R] {
	return func(ctx context.Context, set receiver.Settings, cfg component.Config, next C) (R, error) {
		var limiter *limitedReceiver[P, C, R, traits[P, C, R]]
		var emptyCfg Config
		if lc := cfgf(cfg); lc != emptyCfg {
			limiter = newLimited(next, lc, t)
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
		if limiter == nil {
			return recv, nil
		}
		return t.newReceiver(component.NewComponentImpl(
			func(ctx context.Context, host component.Host) error {
				err1 := limiter.Start(ctx, host)
				err2 := recv.Start(ctx, host)
				return multierr.Append(err1, err2)
			},
			func(ctx context.Context) error {
				err1 := recv.Shutdown(ctx)
				err2 := limiter.Shutdown(ctx)
				return multierr.Append(err1, err2)
			},
		)), nil
	}
}

func NewLimitedFactory(fact xreceiver.Factory, cfgf LimiterConfigurator) xreceiver.Factory {
	return xreceiver.NewFactoryImpl(
		receiver.NewFactoryImpl(
			component.NewFactoryImpl(
				fact.Type,
				fact.CreateDefaultConfig,
			),
			receiver.CreateTracesFunc(limitReceiver(fact.CreateTraces, traceTraits{}, cfgf)),
			fact.TracesStability,
			receiver.CreateMetricsFunc(limitReceiver(fact.CreateMetrics, metricTraits{}, cfgf)),
			fact.MetricsStability,
			receiver.CreateLogsFunc(limitReceiver(fact.CreateLogs, logTraits{}, cfgf)),
			fact.LogsStability,
		),
		xreceiver.CreateProfilesFunc(limitReceiver(fact.CreateProfiles, profileTraits{}, cfgf)),
		fact.ProfilesStability,
	)
}
