// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package service // import "go.opentelemetry.io/collector/service"

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configmapprovider"
	"go.opentelemetry.io/collector/config/configunmarshaler"
	"go.opentelemetry.io/collector/config/experimental/configsource"
)

// ConfigProvider provides the service configuration.
//
// The typical usage is the following:
//
//		cfgProvider.Get(...)
//		cfgProvider.Watch() // wait for an event.
//		cfgProvider.Get(...)
//		cfgProvider.Watch() // wait for an event.
//		// repeat Get/Watch cycle until it is time to shut down the Collector process.
//		cfgProvider.Shutdown()
type ConfigProvider interface {
	// Get returns the service configuration, or error otherwise.
	//
	// Should never be called concurrently with itself, Watch or Shutdown.
	Get(ctx context.Context, factories component.Factories) (*config.Config, error)

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

type configProvider struct {
	locations          []string
	configMapProviders map[string]configmapprovider.Provider
	cfgMapConverters   []config.MapConverterFunc
	configUnmarshaler  configunmarshaler.ConfigUnmarshaler

	sync.Mutex
	closer  configmapprovider.CloseFunc
	watcher chan error
}

// ConfigProviderOptions has options that change the behavior of ConfigProvider
// returned by NewConfigProvider()
type ConfigProviderOptions struct {
	configMapProviders map[string]configmapprovider.Provider
	cfgMapConverters   []config.MapConverterFunc
	configUnmarshaler  configunmarshaler.ConfigUnmarshaler
}

// ConfigProviderOption is an option to change the behavior of ConfigProvider
// returned by NewConfigProvider()
type ConfigProviderOption func(opts *ConfigProviderOptions)

func WithConfigMapProviders(c map[string]configmapprovider.Provider) ConfigProviderOption {
	return func(opts *ConfigProviderOptions) {
		opts.configMapProviders = c
	}
}

func WithCfgMapConverters(c []config.MapConverterFunc) ConfigProviderOption {
	return func(opts *ConfigProviderOptions) {
		opts.cfgMapConverters = c
	}
}

func WithConfigUnmarshaler(c configunmarshaler.ConfigUnmarshaler) ConfigProviderOption {
	return func(opts *ConfigProviderOptions) {
		opts.configUnmarshaler = c
	}
}

// Deprecated: use NewConfigProvider instead
// MustNewConfigProvider returns a new ConfigProvider that provides the configuration:
// * Retrieve the config.Map by merging all retrieved maps from all the configmapprovider.Provider in order.
// * Then applies all the ConfigMapConverterFunc in the given order.
// * Then unmarshalls the final config.Config using the given configunmarshaler.ConfigUnmarshaler.
//
// The `configMapProviders` is a map of pairs <scheme,Provider>.
//
// Notice: This API is experimental.
func MustNewConfigProvider(
	locations []string,
	configMapProviders map[string]configmapprovider.Provider,
	cfgMapConverters []config.MapConverterFunc,
	configUnmarshaler configunmarshaler.ConfigUnmarshaler) ConfigProvider {
	configProvider, err := NewConfigProvider(locations, WithConfigMapProviders(configMapProviders), WithCfgMapConverters(cfgMapConverters), WithConfigUnmarshaler(configUnmarshaler))
	if err != nil {
		panic(err)
	}
	return configProvider
}

// NewConfigProvider returns a new ConfigProvider that provides the configuration:
// * Retrieve the config.Map by merging all retrieved maps from all the configmapprovider.Provider in order.
// * Then applies all the config.MapConverterFunc in the given order.
// * Then unmarshalls the final config.Config using the given configunmarshaler.ConfigUnmarshaler.
//
// The `configMapProviders` is a map of pairs <scheme,Provider>.
//
// Notice: This API is experimental.
func NewConfigProvider(
	locations []string,
	opts ...ConfigProviderOption) (ConfigProvider, error) {

	configOpts := &ConfigProviderOptions{}
	for _, o := range opts {
		o(configOpts)
	}

	if len(locations) == 0 {
		return nil, fmt.Errorf("cannot create ConfigProvider: no locations provided")
	}
	// Safe copy, ensures the slice cannot be changed from the caller.
	locationsCopy := make([]string, len(locations))
	copy(locationsCopy, locations)
	return &configProvider{
		locations:          locationsCopy,
		configMapProviders: configOpts.configMapProviders,
		cfgMapConverters:   configOpts.cfgMapConverters,
		configUnmarshaler:  configOpts.configUnmarshaler,
		watcher:            make(chan error, 1),
	}, nil
}

// Deprecated: use NewDefaultConfigProvider instead
// MustNewDefaultConfigProvider returns the default ConfigProvider, and it creates configuration from a file
// defined by the given configFile and overwrites fields using properties.
func MustNewDefaultConfigProvider(configLocations []string, properties []string) ConfigProvider {
	configProvider, err := NewDefaultConfigProvider(configLocations, properties)
	if err != nil {
		panic(err)
	}
	return configProvider
}

// NewDefaultConfigProvider returns the default ConfigProvider, and it creates configuration from a file
// defined by the given configFile and overwrites fields using properties.
func NewDefaultConfigProvider(configLocations []string, properties []string) (ConfigProvider, error) {
	configMapProvider := map[string]configmapprovider.Provider{
		"file": configmapprovider.NewFile(),
		"env":  configmapprovider.NewEnv(),
	}
	cfgMapConverter := []config.MapConverterFunc{
		configmapprovider.NewOverwritePropertiesConverter(properties),
		configmapprovider.NewExpandConverter(),
	}
	return NewConfigProvider(
		configLocations,
		WithCfgMapConverters(cfgMapConverter),
		WithConfigUnmarshaler(configunmarshaler.NewDefault()),
		WithConfigMapProviders(configMapProvider),
	)
}

func (cm *configProvider) Get(ctx context.Context, factories component.Factories) (*config.Config, error) {
	// First check if already an active watching, close that if any.
	if err := cm.closeIfNeeded(ctx); err != nil {
		return nil, fmt.Errorf("cannot close previous watch: %w", err)
	}

	ret, err := cm.mergeRetrieve(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot retrieve the configuration: %w", err)
	}
	cm.closer = ret.CloseFunc

	// Apply all converters.
	for _, cfgMapConv := range cm.cfgMapConverters {
		if err = cfgMapConv(ctx, ret.Map); err != nil {
			return nil, fmt.Errorf("cannot convert the config.Map: %w", err)
		}
	}

	var cfg *config.Config
	if cfg, err = cm.configUnmarshaler.Unmarshal(ret.Map, factories); err != nil {
		return nil, fmt.Errorf("cannot unmarshal the configuration: %w", err)
	}

	if err = cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

func (cm *configProvider) Watch() <-chan error {
	return cm.watcher
}

func (cm *configProvider) onChange(event *configmapprovider.ChangeEvent) {
	// TODO: Remove check for configsource.ErrSessionClosed when providers updated to not call onChange when closed.
	if event.Error != configsource.ErrSessionClosed {
		cm.watcher <- event.Error
	}
}

func (cm *configProvider) closeIfNeeded(ctx context.Context) error {
	if cm.closer != nil {
		return cm.closer(ctx)
	}
	return nil
}

func (cm *configProvider) Shutdown(ctx context.Context) error {
	close(cm.watcher)

	var errs error
	errs = multierr.Append(errs, cm.closeIfNeeded(ctx))
	for _, p := range cm.configMapProviders {
		errs = multierr.Append(errs, p.Shutdown(ctx))
	}

	return errs
}

// follows drive-letter specification:
// https://tools.ietf.org/id/draft-kerwin-file-scheme-07.html#syntax
var driverLetterRegexp = regexp.MustCompile("^[A-z]:")

func (cm *configProvider) mergeRetrieve(ctx context.Context) (*configmapprovider.Retrieved, error) {
	var closers []configmapprovider.CloseFunc
	retCfgMap := config.NewMap()
	for _, location := range cm.locations {
		// For backwards compatibility:
		// - empty url scheme means "file".
		// - "^[A-z]:" also means "file"
		scheme := "file"
		if idx := strings.Index(location, ":"); idx != -1 && !driverLetterRegexp.MatchString(location) {
			scheme = location[:idx]
		} else {
			location = scheme + ":" + location
		}
		p, ok := cm.configMapProviders[scheme]
		if !ok {
			return nil, fmt.Errorf("scheme %v is not supported for location %v", scheme, location)
		}
		retr, err := p.Retrieve(ctx, location, cm.onChange)
		if err != nil {
			return nil, err
		}
		if err = retCfgMap.Merge(retr.Map); err != nil {
			return nil, err
		}
		if retr.CloseFunc != nil {
			closers = append(closers, retr.CloseFunc)
		}
	}
	return &configmapprovider.Retrieved{
		Map: retCfgMap,
		CloseFunc: func(ctxF context.Context) error {
			var err error
			for _, ret := range closers {
				err = multierr.Append(err, ret(ctxF))
			}
			return err
		},
	}, nil
}
