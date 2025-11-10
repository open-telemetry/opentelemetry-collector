// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:generate mdatagen metadata.yaml

package memorylimiterprocessor // import "go.opentelemetry.io/collector/processor/memorylimiterprocessor"

import (
	"context"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/internal/memorylimiter"
	"go.opentelemetry.io/collector/internal/telemetry"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/memorylimiterprocessor/internal/metadata"
	"go.opentelemetry.io/collector/processor/processorhelper"
	"go.opentelemetry.io/collector/processor/processorhelper/xprocessorhelper"
	"go.opentelemetry.io/collector/processor/xprocessor"
)

var processorCapabilities = consumer.Capabilities{MutatesData: false}

type factory struct {
	// memoryLimiters stores memoryLimiter instances with unique configs that multiple processors can reuse.
	// This avoids running multiple memory checks (ie: GC) for every processor using the same processor config.
	memoryLimiters map[component.Config]*memoryLimiterProcessor
	lock           sync.Mutex
}

// NewFactory returns a new factory for the Memory Limiter processor.
func NewFactory() xprocessor.Factory {
	f := &factory{
		memoryLimiters: map[component.Config]*memoryLimiterProcessor{},
	}
	return xprocessor.NewFactory(
		metadata.Type,
		createDefaultConfig,
		xprocessor.WithTraces(f.createTraces, metadata.TracesStability),
		xprocessor.WithMetrics(f.createMetrics, metadata.MetricsStability),
		xprocessor.WithLogs(f.createLogs, metadata.LogsStability),
		xprocessor.WithProfiles(f.createProfiles, metadata.ProfilesStability))
}

// CreateDefaultConfig creates the default configuration for processor. Notice
// that the default configuration is expected to fail for this processor.
func createDefaultConfig() component.Config {
	return memorylimiter.NewDefaultConfig()
}

func (f *factory) createTraces(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (processor.Traces, error) {
	memLimiter, err := f.getMemoryLimiter(set, cfg)
	if err != nil {
		return nil, err
	}
	return processorhelper.NewTraces(ctx, set, cfg, nextConsumer,
		memLimiter.processTraces,
		processorhelper.WithCapabilities(processorCapabilities),
		processorhelper.WithStart(memLimiter.start),
		processorhelper.WithShutdown(memLimiter.shutdown))
}

func (f *factory) createMetrics(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (processor.Metrics, error) {
	memLimiter, err := f.getMemoryLimiter(set, cfg)
	if err != nil {
		return nil, err
	}
	return processorhelper.NewMetrics(ctx, set, cfg, nextConsumer,
		memLimiter.processMetrics,
		processorhelper.WithCapabilities(processorCapabilities),
		processorhelper.WithStart(memLimiter.start),
		processorhelper.WithShutdown(memLimiter.shutdown))
}

func (f *factory) createLogs(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (processor.Logs, error) {
	memLimiter, err := f.getMemoryLimiter(set, cfg)
	if err != nil {
		return nil, err
	}
	return processorhelper.NewLogs(ctx, set, cfg, nextConsumer,
		memLimiter.processLogs,
		processorhelper.WithCapabilities(processorCapabilities),
		processorhelper.WithStart(memLimiter.start),
		processorhelper.WithShutdown(memLimiter.shutdown))
}

func (f *factory) createProfiles(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer xconsumer.Profiles,
) (xprocessor.Profiles, error) {
	memLimiter, err := f.getMemoryLimiter(set, cfg)
	if err != nil {
		return nil, err
	}

	return xprocessorhelper.NewProfiles(
		ctx,
		set,
		cfg,
		nextConsumer,
		memLimiter.processProfiles,
		xprocessorhelper.WithCapabilities(processorCapabilities),
		xprocessorhelper.WithStart(memLimiter.start),
		xprocessorhelper.WithShutdown(memLimiter.shutdown),
	)
}

// getMemoryLimiter checks if we have a cached memoryLimiter with a specific config,
// otherwise initialize and add one to the store.
func (f *factory) getMemoryLimiter(set processor.Settings, cfg component.Config) (*memoryLimiterProcessor, error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	if memLimiter, ok := f.memoryLimiters[cfg]; ok {
		return memLimiter, nil
	}

	set.TelemetrySettings = telemetry.DropInjectedAttributes(
		set.TelemetrySettings,
		telemetry.SignalKey,
		telemetry.PipelineIDKey,
		telemetry.ComponentIDKey,
	)
	set.Logger.Debug("created singleton logger")

	memLimiter, err := newMemoryLimiterProcessor(set, cfg.(*Config))
	if err != nil {
		return nil, err
	}

	f.memoryLimiters[cfg] = memLimiter
	return memLimiter, nil
}
