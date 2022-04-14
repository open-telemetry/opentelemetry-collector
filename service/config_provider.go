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
	"go.opentelemetry.io/collector/config/configunmarshaler"
	"go.opentelemetry.io/collector/config/experimental/configsource"
	"go.opentelemetry.io/collector/config/mapconverter/expandmapconverter"
	"go.opentelemetry.io/collector/config/mapprovider/envmapprovider"
	"go.opentelemetry.io/collector/config/mapprovider/filemapprovider"
	"go.opentelemetry.io/collector/config/mapprovider/yamlmapprovider"
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
	locations           []string
	configMapProviders  map[string]config.MapProvider
	configMapConverters []config.MapConverterFunc
	configUnmarshaler   configunmarshaler.ConfigUnmarshaler

	sync.Mutex
	closer  config.CloseFunc
	watcher chan error
}

// ConfigProviderSettings are the settings to configure the behavior of the ConfigProvider.
type ConfigProviderSettings struct {
	// Locations from where the config.Map is retrieved, and merged in the given order.
	// It is required to have at least one location.
	Locations []string

	// MapProviders is a map of pairs <scheme, config.MapProvider>.
	// It is required to have at least one config.MapProvider.
	MapProviders map[string]config.MapProvider

	// MapConverters is a slice of config.MapConverterFunc.
	MapConverters []config.MapConverterFunc

	// The configunmarshaler.ConfigUnmarshaler to be used to unmarshal the config.Map into config.Config.
	// It is required to not be nil, use configunmarshaler.NewDefault() by default.
	Unmarshaler configunmarshaler.ConfigUnmarshaler
}

func newDefaultConfigProviderSettings(locations []string) ConfigProviderSettings {
	return ConfigProviderSettings{
		Locations:     locations,
		MapProviders:  makeConfigMapProviderMap(filemapprovider.New(), envmapprovider.New(), yamlmapprovider.New()),
		MapConverters: []config.MapConverterFunc{expandmapconverter.New()},
		Unmarshaler:   configunmarshaler.NewDefault(),
	}
}

// NewConfigProvider returns a new ConfigProvider that provides the configuration:
// * Retrieve the config.Map by merging all retrieved maps from the given `locations` in order.
// * Then applies all the config.MapConverterFunc in the given order.
// * Then unmarshals the config.Map into the service Config.
func NewConfigProvider(set ConfigProviderSettings) (ConfigProvider, error) {
	if len(set.Locations) == 0 {
		return nil, fmt.Errorf("cannot create ConfigProvider: no Locations")
	}

	if len(set.MapProviders) == 0 {
		return nil, fmt.Errorf("cannot create ConfigProvider: no MapProviders")
	}

	// Safe copy, ensures the slices and maps cannot be changed from the caller.
	locationsCopy := make([]string, len(set.Locations))
	copy(locationsCopy, set.Locations)
	mapProvidersCopy := make(map[string]config.MapProvider, len(set.MapProviders))
	for k, v := range set.MapProviders {
		mapProvidersCopy[k] = v
	}
	mapConvertersCopy := make([]config.MapConverterFunc, len(set.MapConverters))
	copy(mapConvertersCopy, set.MapConverters)

	return &configProvider{
		locations:           locationsCopy,
		configMapProviders:  mapProvidersCopy,
		configMapConverters: mapConvertersCopy,
		configUnmarshaler:   set.Unmarshaler,
		watcher:             make(chan error, 1),
	}, nil
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
	for _, cfgMapConv := range cm.configMapConverters {
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

func (cm *configProvider) onChange(event *config.ChangeEvent) {
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

func (cm *configProvider) mergeRetrieve(ctx context.Context) (*config.Retrieved, error) {
	var closers []config.CloseFunc
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
	return &config.Retrieved{
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

func makeConfigMapProviderMap(providers ...config.MapProvider) map[string]config.MapProvider {
	ret := make(map[string]config.MapProvider, len(providers))
	for _, provider := range providers {
		ret[provider.Scheme()] = provider
	}
	return ret
}
