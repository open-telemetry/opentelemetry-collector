// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol // import "go.opentelemetry.io/collector/otelcol"

import (
	"context"
	"fmt"

	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
)

// ConfigProvider provides the service configuration.
//
// The typical usage is the following:
//
//	cfgProvider.Get(...)
//	cfgProvider.Watch() // wait for an event.
//	cfgProvider.Get(...)
//	cfgProvider.Watch() // wait for an event.
//	// repeat Get/Watch cycle until it is time to shut down the Collector process.
//	cfgProvider.Shutdown()
type ConfigProvider struct {
	mapResolver *confmap.Resolver
}

// ConfigProviderSettings are the settings to configure the behavior of the ConfigProvider.
type ConfigProviderSettings struct {
	// ResolverSettings are the settings to configure the behavior of the confmap.Resolver.
	ResolverSettings confmap.ResolverSettings
	// prevent unkeyed literal initialization
	_ struct{}
}

// NewConfigProvider returns a new ConfigProvider that provides the service configuration:
// * Initially it resolves the "configuration map":
//   - Retrieve the confmap.Conf by merging all retrieved maps from the given `locations` in order.
//   - Then applies all the confmap.Converter in the given order.
//
// * Then unmarshalls the confmap.Conf into the service Config.
func NewConfigProvider(set ConfigProviderSettings) (*ConfigProvider, error) {
	mr, err := confmap.NewResolver(set.ResolverSettings)
	if err != nil {
		return nil, err
	}

	return &ConfigProvider{
		mapResolver: mr,
	}, nil
}

// Get returns the service configuration, or error otherwise.
//
// Should never be called concurrently with itself, Watch or Shutdown.
func (cm *ConfigProvider) Get(ctx context.Context, factories Factories) (*Config, error) {
	conf, err := cm.mapResolver.Resolve(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve the configuration: %w", err)
	}

	var cfg *configSettings
	if cfg, err = unmarshal(conf, factories); err != nil {
		return nil, fmt.Errorf("cannot unmarshal the configuration: %w", err)
	}

	var logLevels map[component.Kind]map[component.ID]zapcore.Level
	for _, entry := range []struct {
		kind   component.Kind
		levels map[component.ID]zapcore.Level
	}{
		{component.KindReceiver, cfg.Receivers.LogLevels()},
		{component.KindProcessor, cfg.Processors.LogLevels()},
		{component.KindExporter, cfg.Exporters.LogLevels()},
		{component.KindConnector, cfg.Connectors.LogLevels()},
		{component.KindExtension, cfg.Extensions.LogLevels()},
	} {
		if len(entry.levels) > 0 {
			if logLevels == nil {
				logLevels = make(map[component.Kind]map[component.ID]zapcore.Level)
			}
			logLevels[entry.kind] = entry.levels
		}
	}

	return &Config{
		Receivers:          cfg.Receivers.Configs(),
		Processors:         cfg.Processors.Configs(),
		Exporters:          cfg.Exporters.Configs(),
		Connectors:         cfg.Connectors.Configs(),
		Extensions:         cfg.Extensions.Configs(),
		Service:            cfg.Service,
		ComponentLogLevels: logLevels,
	}, nil
}

// Watch blocks until any configuration change was detected or an unrecoverable error
// happened during monitoring the configuration changes.
//
// Error is nil if the configuration is changed and needs to be re-fetched. Any non-nil
// error indicates that there was a problem with watching the config changes.
//
// Should never be called concurrently with itself or Get.
func (cm *ConfigProvider) Watch() <-chan error {
	return cm.mapResolver.Watch()
}

// Shutdown signals that the provider is no longer in use and the that should close
// and release any resources that it may have created.
//
// This function must terminate the Watch channel.
//
// Should never be called concurrently with itself or Get.
func (cm *ConfigProvider) Shutdown(ctx context.Context) error {
	return cm.mapResolver.Shutdown(ctx)
}
