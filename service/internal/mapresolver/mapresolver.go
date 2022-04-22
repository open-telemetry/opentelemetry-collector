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

package mapresolver // import "go.opentelemetry.io/collector/service/internal/mapresolver"

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/experimental/configsource"
	"go.opentelemetry.io/collector/config/mapconverter/expandmapconverter"
	"go.opentelemetry.io/collector/config/mapprovider/envmapprovider"
	"go.opentelemetry.io/collector/config/mapprovider/filemapprovider"
	"go.opentelemetry.io/collector/config/mapprovider/yamlmapprovider"
)

// follows drive-letter specification:
// https://tools.ietf.org/id/draft-kerwin-file-scheme-07.html#syntax
var driverLetterRegexp = regexp.MustCompile("^[A-z]:")

// Settings are the settings to configure the MapResolver.
type Settings struct {
	// Locations from where the config.Map is retrieved, and merged in the given order.
	// It is required to have at least one location.
	Locations []string

	// MapProviders is a map of pairs <scheme, config.MapProvider>.
	// It is required to have at least one config.MapProvider.
	MapProviders map[string]config.MapProvider

	// MapConverters is a slice of config.MapConverterFunc.
	MapConverters []config.MapConverterFunc
}

func NewDefaultSettings(locations []string) *Settings {
	return &Settings{
		Locations:     locations,
		MapProviders:  makeMapProvidersMap(filemapprovider.New(), envmapprovider.New(), yamlmapprovider.New()),
		MapConverters: []config.MapConverterFunc{expandmapconverter.New()},
	}
}

// MapResolver resolves a configuration as a config.Map.
type MapResolver struct {
	locations           []string
	configMapProviders  map[string]config.MapProvider
	configMapConverters []config.MapConverterFunc

	sync.Mutex
	closers []config.CloseFunc
	watcher chan error
}

// NewMapResolver returns a new MapResolver that resolves configuration from multiple locations.
//
// To resolve a configuration the following steps will happen:
//   1. Retrieves individual configurations from all given "locations", and merge them in the retrieve order.
//   2. Once the config.Map is merged, apply the converters in the given order.
//
// After the configuration was resolved the `MapResolver` can be used as a single point to watch for updates in
// the configuration data retrieved via the config providers used to process the "initial" configuration and to generate
// the "effective" one. The typical usage is the following:
//
//		mapResolver.Resolve(ctx)
//		mapResolver.Watch() // wait for an event.
//		mapResolver.Resolve(ctx)
//		mapResolver.Watch() // wait for an event.
//		// repeat Resolve/Watch cycle until it is time to shut down the Collector process.
//		mapResolver.Shutdown(ctx)
func NewMapResolver(set *Settings) (*MapResolver, error) {
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

	return &MapResolver{
		locations:           locationsCopy,
		configMapProviders:  mapProvidersCopy,
		configMapConverters: mapConvertersCopy,
		watcher:             make(chan error, 1),
	}, nil
}

// Resolve returns the configuration as a config.Map, or error otherwise.
//
// Should never be called concurrently with itself, Watch or Shutdown.
func (mr *MapResolver) Resolve(ctx context.Context) (*config.Map, error) {
	// First check if already an active watching, close that if any.
	if err := mr.closeIfNeeded(ctx); err != nil {
		return nil, fmt.Errorf("cannot close previous watch: %w", err)
	}

	// Retrieves individual configurations from all locations in the given order, and merge them in retMap.
	retMap := config.NewMap()
	for _, location := range mr.locations {
		// For backwards compatibility:
		// - empty url scheme means "file".
		// - "^[A-z]:" also means "file"
		scheme := "file"
		if idx := strings.Index(location, ":"); idx != -1 && !driverLetterRegexp.MatchString(location) {
			scheme = location[:idx]
		} else {
			location = scheme + ":" + location
		}
		p, ok := mr.configMapProviders[scheme]
		if !ok {
			return nil, fmt.Errorf("scheme %v is not supported for location %v", scheme, location)
		}
		ret, err := p.Retrieve(ctx, location, mr.onChange)
		if err != nil {
			return nil, err
		}
		retCfgMap, err := ret.AsMap()
		if err != nil {
			return nil, err
		}
		if err = retMap.Merge(retCfgMap); err != nil {
			return nil, err
		}
		mr.closers = append(mr.closers, ret.Close)
	}

	// Apply the converters in the given order.
	for _, cfgMapConv := range mr.configMapConverters {
		if err := cfgMapConv(ctx, retMap); err != nil {
			return nil, fmt.Errorf("cannot convert the config.Map: %w", err)
		}
	}

	return retMap, nil
}

// Watch blocks until any configuration change was detected or an unrecoverable error
// happened during monitoring the configuration changes.
//
// Error is nil if the configuration is changed and needs to be re-fetched. Any non-nil
// error indicates that there was a problem with watching the configuration changes.
//
// Should never be called concurrently with itself or Get.
func (mr *MapResolver) Watch() <-chan error {
	return mr.watcher
}

// Shutdown signals that the provider is no longer in use and the that should close
// and release any resources that it may have created. It terminates the Watch channel.
//
// Should never be called concurrently with itself or Get.
func (mr *MapResolver) Shutdown(ctx context.Context) error {
	close(mr.watcher)

	var errs error
	errs = multierr.Append(errs, mr.closeIfNeeded(ctx))
	for _, p := range mr.configMapProviders {
		errs = multierr.Append(errs, p.Shutdown(ctx))
	}

	return errs
}

func (mr *MapResolver) onChange(event *config.ChangeEvent) {
	// TODO: Remove check for configsource.ErrSessionClosed when providers updated to not call onChange when closed.
	if event.Error != configsource.ErrSessionClosed {
		mr.watcher <- event.Error
	}
}

func (mr *MapResolver) closeIfNeeded(ctx context.Context) error {
	var err error
	for _, ret := range mr.closers {
		err = multierr.Append(err, ret(ctx))
	}
	return err
}

func makeMapProvidersMap(providers ...config.MapProvider) map[string]config.MapProvider {
	ret := make(map[string]config.MapProvider, len(providers))
	for _, provider := range providers {
		ret[provider.Scheme()] = provider
	}
	return ret
}
