// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package limiterhelper // import "go.opentelemetry.io/collector/extension/extensionlimiter/limiterhelper"

import (
	"context"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/extension/extensionlimiter"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// Consumer is a builder for creating wrapped consumers with resource limiters
//
// This supports limiting by request_count, request_items, and
// memory_size weight keys.
//
// The network_bytes weight key not supported because that information
// is not available from the pdata object.
type limiter struct {
	requestItemsLimiter extensionlimiter.ResourceLimiter
	memorySizeLimiter   extensionlimiter.ResourceLimiter
	requestCountLimiter extensionlimiter.ResourceLimiter
}

// config stores configuration from Options.
type config struct {
	requestItemsLimiter bool
	memorySizeLimiter   bool
	requestCountLimiter bool
}

// Option represents the consumer options
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (of optionFunc) apply(cfg *config) {
	of(cfg)
}

// WithRequestCountLimit configures the consumer to limit based on request count.
func WithRequestCountLimit() Option {
	return optionFunc(func(c *config) {
		c.requestCountLimiter = true
	})
}

// WithRequestItemsLimit configures the consumer to limit based on item counts.
func WithRequestItemsLimit() Option {
	return optionFunc(func(c *config) {
		c.requestItemsLimiter = true
	})
}

// WithMemorySizeLimit configures the consumer to limit based on memory size.
func WithMemorySizeLimit() Option {
	return optionFunc(func(c *config) {
		c.memorySizeLimiter = true
	})
}

func newLimiter(provider extensionlimiter.Provider, options ...Option) *limiter {
	cfg := &config{}
	for _, option := range options {
		option.apply(cfg)
	}
	c := &limiter{}
	if cfg.requestCountLimiter {
		c.requestCountLimiter = provider.ResourceLimiter(extensionlimiter.WeightKeyRequestCount)
	}
	if cfg.requestItemsLimiter {
		c.requestItemsLimiter = provider.ResourceLimiter(extensionlimiter.WeightKeyRequestItems)
	}
	if cfg.memorySizeLimiter {
		c.memorySizeLimiter = provider.ResourceLimiter(extensionlimiter.WeightKeyMemorySize)
	}
	return c
}

// WrapTraces wraps a traces consumer with resource limiters
func WrapTraces(provider extensionlimiter.Provider, nextConsumer consumer.Traces, options ...Option) consumer.Traces {
	limiter := newLimiter(provider, options...)

	if limiter.requestItemsLimiter == nil && limiter.memorySizeLimiter == nil && limiter.requestCountLimiter == nil {
		return nextConsumer
	}
	return &tracesConsumer{
		nextConsumer: nextConsumer,
		limiter:      limiter,
	}
}

// WrapMetrics wraps a metrics consumer with resource limiters
func WrapMetrics(provider extensionlimiter.Provider, nextConsumer consumer.Metrics, options ...Option) consumer.Metrics {
	limiter := newLimiter(provider, options...)

	if limiter.requestItemsLimiter == nil && limiter.memorySizeLimiter == nil && limiter.requestCountLimiter == nil {
		return nextConsumer
	}
	return &metricsConsumer{
		nextConsumer: nextConsumer,
		limiter:      limiter,
	}
}

// WrapLogs wraps a logs consumer with resource limiters
func WrapLogs(provider extensionlimiter.Provider, nextConsumer consumer.Logs, options ...Option) consumer.Logs {
	limiter := newLimiter(provider, options...)
	if limiter.requestItemsLimiter == nil && limiter.memorySizeLimiter == nil && limiter.requestCountLimiter == nil {
		return nextConsumer
	}
	return &logsConsumer{
		nextConsumer: nextConsumer,
		limiter:      limiter,
	}
}

// WrapProfiles wraps a profiles consumer with resource limiters
func WrapProfiles(provider extensionlimiter.Provider, nextConsumer xconsumer.Profiles, options ...Option) xconsumer.Profiles {
	limiter := newLimiter(provider, options...)

	if limiter.requestItemsLimiter == nil && limiter.memorySizeLimiter == nil && limiter.requestCountLimiter == nil {
		return nextConsumer
	}
	return &profilesConsumer{
		nextConsumer: nextConsumer,
		limiter:      limiter,
	}
}

// Signal-specific consumer implementations
type tracesConsumer struct {
	nextConsumer consumer.Traces
	*limiter
}

func (tc *tracesConsumer) Capabilities() consumer.Capabilities {
	return tc.nextConsumer.Capabilities()
}

func (tc *tracesConsumer) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	numSpans := td.SpanCount()
	if numSpans == 0 {
		return tc.nextConsumer.ConsumeTraces(ctx, td)
	}

	// Apply the request count limiter if available
	if tc.requestCountLimiter != nil {
		release, err := tc.requestCountLimiter.Acquire(ctx, 1)
		defer release()
		if err != nil {
			return err
		}
	}

	// Apply the items limiter if available
	if tc.requestItemsLimiter != nil {
		release, err := tc.requestItemsLimiter.Acquire(ctx, uint64(numSpans))
		defer release()
		if err != nil {
			return err
		}
	}

	// Apply the memory size limiter if available
	if tc.memorySizeLimiter != nil {
		// Get the marshaled size of the request as a proxy for memory size
		var sizer ptrace.ProtoMarshaler
		size := sizer.TracesSize(td)
		release, err := tc.memorySizeLimiter.Acquire(ctx, uint64(size))
		defer release()
		if err != nil {
			return err
		}
	}

	return tc.nextConsumer.ConsumeTraces(ctx, td)
}

type metricsConsumer struct {
	nextConsumer consumer.Metrics
	*limiter
}

func (mc *metricsConsumer) Capabilities() consumer.Capabilities {
	return mc.nextConsumer.Capabilities()
}

func (mc *metricsConsumer) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	dataPointCount := md.DataPointCount()
	if dataPointCount == 0 {
		return mc.nextConsumer.ConsumeMetrics(ctx, md)
	}

	// Apply the request count limiter if available
	if mc.requestCountLimiter != nil {
		release, err := mc.requestCountLimiter.Acquire(ctx, 1)
		defer release()
		if err != nil {
			return err
		}
	}

	// Apply the items limiter if available
	if mc.requestItemsLimiter != nil {
		release, err := mc.requestItemsLimiter.Acquire(ctx, uint64(dataPointCount))
		defer release()
		if err != nil {
			return err
		}
	}

	// Apply the memory size limiter if available
	if mc.memorySizeLimiter != nil {
		var sizer pmetric.ProtoMarshaler
		size := sizer.MetricsSize(md)
		release, err := mc.memorySizeLimiter.Acquire(ctx, uint64(size))
		defer release()
		if err != nil {
			return err
		}
	}

	return mc.nextConsumer.ConsumeMetrics(ctx, md)
}

type logsConsumer struct {
	nextConsumer consumer.Logs
	*limiter
}

func (lc *logsConsumer) Capabilities() consumer.Capabilities {
	return lc.nextConsumer.Capabilities()
}

func (lc *logsConsumer) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	numRecords := ld.LogRecordCount()
	if numRecords == 0 {
		return lc.nextConsumer.ConsumeLogs(ctx, ld)
	}

	// Apply the request count limiter if available
	if lc.requestCountLimiter != nil {
		release, err := lc.requestCountLimiter.Acquire(ctx, 1)
		defer release()
		if err != nil {
			return err
		}
	}

	// Apply the items limiter if available
	if lc.requestItemsLimiter != nil {
		release, err := lc.requestItemsLimiter.Acquire(ctx, uint64(numRecords))
		defer release()
		if err != nil {
			return err
		}
	}

	// Apply the memory size limiter if available
	if lc.memorySizeLimiter != nil {
		var sizer plog.ProtoMarshaler
		size := sizer.LogsSize(ld)
		release, err := lc.memorySizeLimiter.Acquire(ctx, uint64(size))
		defer release()
		if err != nil {
			return err
		}
	}

	return lc.nextConsumer.ConsumeLogs(ctx, ld)
}

type profilesConsumer struct {
	nextConsumer xconsumer.Profiles
	*limiter
}

func (pc *profilesConsumer) ConsumeProfiles(ctx context.Context, pd pprofile.Profiles) error {
	numProfiles := pd.SampleCount()
	if numProfiles == 0 {
		return pc.nextConsumer.ConsumeProfiles(ctx, pd)
	}

	// Apply the request count limiter if available
	if pc.requestCountLimiter != nil {
		release, err := pc.requestCountLimiter.Acquire(ctx, 1)
		defer release()
		if err != nil {
			return err
		}
	}

	// Apply the items limiter if available
	if pc.requestItemsLimiter != nil {
		release, err := pc.requestItemsLimiter.Acquire(ctx, uint64(numProfiles))
		defer release()
		if err != nil {
			return err
		}
	}

	// Apply the memory size limiter if available
	if pc.memorySizeLimiter != nil {
		var sizer pprofile.ProtoMarshaler
		size := sizer.ProfilesSize(pd)
		release, err := pc.memorySizeLimiter.Acquire(ctx, uint64(size))
		defer release()
		if err != nil {
			return err
		}
	}

	return pc.nextConsumer.ConsumeProfiles(ctx, pd)
}

func (pc *profilesConsumer) Capabilities() consumer.Capabilities {
	return pc.nextConsumer.Capabilities()
}
