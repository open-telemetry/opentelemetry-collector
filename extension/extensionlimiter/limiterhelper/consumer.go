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
type Consumer struct {
	requestItemsLimiter extensionlimiter.ResourceLimiter
	memorySizeLimiter   extensionlimiter.ResourceLimiter
	requestCountLimiter extensionlimiter.ResourceLimiter
}

// Config stores configuration from Options.
type Config struct {
	requestItemsLimiter bool
	memorySizeLimiter   bool
	requestCountLimiter bool
}

// Option represents the consumer options
type Option func(*Config)

// WithRequestCountLimit configures the consumer to limit based on request count.
func WithRequestCountLimit() Option {
	return func(c *Config) {
		c.requestCountLimiter = true
	}
}

// WithRequestItemsLimit configures the consumer to limit based on item counts.
func WithRequestItemsLimit() Option {
	return func(c *Config) {
		c.requestItemsLimiter = true
	}
}

// WithMemorySizeLimit configures the consumer to limit based on memory size.
func WithMemorySizeLimit() Option {
	return func(c *Config) {
		c.memorySizeLimiter = true
	}
}

// NewConsumer creates a new limiterhelper Consumer
func NewConsumer(provider extensionlimiter.Provider, options ...Option) *Consumer {
	cfg := &Config{}
	for _, option := range options {
		option(cfg)
	}
	c := &Consumer{}
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
func (c *Consumer) WrapTraces(nextConsumer consumer.Traces) consumer.Traces {
	if c.requestItemsLimiter == nil && c.memorySizeLimiter == nil && c.requestCountLimiter == nil {
		return nextConsumer
	}
	return &tracesConsumer{
		nextConsumer: nextConsumer,
		Consumer:     c,
	}
}

// WrapMetrics wraps a metrics consumer with resource limiters
func (c *Consumer) WrapMetrics(nextConsumer consumer.Metrics) consumer.Metrics {
	if c.requestItemsLimiter == nil && c.memorySizeLimiter == nil && c.requestCountLimiter == nil {
		return nextConsumer
	}
	return &metricsConsumer{
		nextConsumer: nextConsumer,
		Consumer:     c,
	}
}

// WrapLogs wraps a logs consumer with resource limiters
func (c *Consumer) WrapLogs(nextConsumer consumer.Logs) consumer.Logs {
	if c.requestItemsLimiter == nil && c.memorySizeLimiter == nil && c.requestCountLimiter == nil {
		return nextConsumer
	}
	return &logsConsumer{
		nextConsumer: nextConsumer,
		Consumer:     c,
	}
}

// WrapProfiles wraps a profiles consumer with resource limiters
func (c *Consumer) WrapProfiles(nextConsumer xconsumer.Profiles) xconsumer.Profiles {
	if c.requestItemsLimiter == nil && c.memorySizeLimiter == nil && c.requestCountLimiter == nil {
		return nextConsumer
	}
	return &profilesConsumer{
		nextConsumer: nextConsumer,
		Consumer:     c,
	}
}

// Signal-specific consumer implementations
type tracesConsumer struct {
	nextConsumer consumer.Traces
	*Consumer
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
	*Consumer
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
	*Consumer
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
	*Consumer
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
