// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol // import "go.opentelemetry.io/collector/otelcol"

import (
	"context"
	"fmt"
	"strings"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/internal/globalgates"
)

var (
	strictlyTypedMessageCoda = `Hint: Temporarily restore the previous behavior by disabling 
      the ` + fmt.Sprintf("`%s`", globalgates.StrictlyTypedInputID) + ` feature gate. More details at:
      https://github.com/open-telemetry/opentelemetry-collector/issues/10552`
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
type ConfigProvider interface {
	// Get returns the service configuration, or error otherwise.
	//
	// Should never be called concurrently with itself, Watch or Shutdown.
	Get(ctx context.Context, factories Factories) (*Config, error)

	// Watch blocks until any configuration change was detected or an unrecoverable error
	// happened during monitoring the configuration changes.
	//
	// Error is nil if the configuration is changed and needs to be re-fetched. Any non-nil
	// error indicates that there was a problem with watching the config changes.
	//
	// Should never be called concurrently with itself or Get.
	Watch() <-chan error

	// Shutdown signals that the provider is no longer in use and the that should close
	// and release any resources that it may have created.
	//
	// This function must terminate the Watch channel.
	//
	// Should never be called concurrently with itself or Get.
	Shutdown(ctx context.Context) error
}

// ConfmapProvider is an optional interface to be implemented by ConfigProviders
// to provide confmap.Conf objects representing a marshaled version of the
// Collector's configuration.
//
// The purpose of this interface is that otelcol.ConfigProvider structs do not
// necessarily need to use confmap.Conf as their underlying config structure.
//
// Deprecated: [v0.105.0] This interface is deprecated. otelcol.Collector will now obtain
// a confmap.Conf object from the unmarshaled config itself.
type ConfmapProvider interface {
	// GetConfmap resolves the Collector's configuration and provides it as a confmap.Conf object.
	//
	// Should never be called concurrently with itself or any ConfigProvider method.
	GetConfmap(ctx context.Context) (*confmap.Conf, error)
}

type configProvider struct {
	mapResolver *confmap.Resolver
}

var _ ConfigProvider = (*configProvider)(nil)

// ConfigProviderSettings are the settings to configure the behavior of the ConfigProvider.
type ConfigProviderSettings struct {
	// ResolverSettings are the settings to configure the behavior of the confmap.Resolver.
	ResolverSettings confmap.ResolverSettings
}

// NewConfigProvider returns a new ConfigProvider that provides the service configuration:
// * Initially it resolves the "configuration map":
//   - Retrieve the confmap.Conf by merging all retrieved maps from the given `locations` in order.
//   - Then applies all the confmap.Converter in the given order.
//
// * Then unmarshalls the confmap.Conf into the service Config.
func NewConfigProvider(set ConfigProviderSettings) (ConfigProvider, error) {
	mr, err := confmap.NewResolver(set.ResolverSettings)
	if err != nil {
		return nil, err
	}

	return &configProvider{
		mapResolver: mr,
	}, nil
}

func (cm *configProvider) Get(ctx context.Context, factories Factories) (*Config, error) {
	conf, err := cm.mapResolver.Resolve(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve the configuration: %w", err)
	}

	var cfg *configSettings
	if cfg, err = unmarshal(conf, factories); err != nil {
		err = fmt.Errorf("cannot unmarshal the configuration: %w", err)

		if globalgates.StrictlyTypedInputGate.IsEnabled() {
			var shouldAddCoda bool
			for _, errorStr := range []string{
				"got unconvertible type",      // https://github.com/mitchellh/mapstructure/blob/8508981/mapstructure.go#L610
				"source data must be",         // https://github.com/mitchellh/mapstructure/blob/8508981/mapstructure.go#L1114
				"expected a map, got 'slice'", // https://github.com/mitchellh/mapstructure/blob/8508981/mapstructure.go#L831
			} {
				shouldAddCoda = strings.Contains(err.Error(), errorStr)
				if shouldAddCoda {
					break
				}
			}
			if shouldAddCoda {
				err = fmt.Errorf("%w\n\n%s", err, strictlyTypedMessageCoda)
			}
		}

		return nil, err
	}

	return &Config{
		Receivers:  cfg.Receivers.Configs(),
		Processors: cfg.Processors.Configs(),
		Exporters:  cfg.Exporters.Configs(),
		Connectors: cfg.Connectors.Configs(),
		Extensions: cfg.Extensions.Configs(),
		Service:    cfg.Service,
	}, nil
}

func (cm *configProvider) Watch() <-chan error {
	return cm.mapResolver.Watch()
}

func (cm *configProvider) Shutdown(ctx context.Context) error {
	return cm.mapResolver.Shutdown(ctx)
}

// Deprecated: [v0.105.0] Call `(*confmap.Conf).Marshal(*otelcol.Config)` to get
// the Collector's configuration instead.
func (cm *configProvider) GetConfmap(ctx context.Context) (*confmap.Conf, error) {
	conf, err := cm.mapResolver.Resolve(ctx)

	if err != nil {
		return nil, fmt.Errorf("cannot resolve the configuration: %w", err)
	}

	return conf, nil
}
