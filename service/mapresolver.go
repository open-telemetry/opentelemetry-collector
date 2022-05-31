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
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/config/experimental/configsource"
	"go.opentelemetry.io/collector/confmap"
)

// follows drive-letter specification:
// https://tools.ietf.org/id/draft-kerwin-file-scheme-07.html#syntax
var driverLetterRegexp = regexp.MustCompile("^[A-z]:")

// mapResolver resolves a configuration as a confmap.Conf.
type mapResolver struct {
	uris       []string
	providers  map[string]confmap.Provider
	converters []confmap.Converter

	sync.Mutex
	closers []confmap.CloseFunc
	watcher chan error
}

// newMapResolver returns a new mapResolver that resolves configuration from multiple URIs.
//
// To resolve a configuration the following steps will happen:
//   1. Retrieves individual configurations from all given "URIs", and merge them in the retrieve order.
//   2. Once the confmap.Conf is merged, apply the converters in the given order.
//
// After the configuration was resolved the `mapResolver` can be used as a single point to watch for updates in
// the configuration data retrieved via the config providers used to process the "initial" configuration and to generate
// the "effective" one. The typical usage is the following:
//
//		mapResolver.Resolve(ctx)
//		mapResolver.Watch() // wait for an event.
//		mapResolver.Resolve(ctx)
//		mapResolver.Watch() // wait for an event.
//		// repeat Resolve/Watch cycle until it is time to shut down the Collector process.
//		mapResolver.Shutdown(ctx)
//
// `uri` must follow the "<scheme>:<opaque_data>" format. This format is compatible with the URI definition
// (see https://datatracker.ietf.org/doc/html/rfc3986). An empty "<scheme>" defaults to "file" schema.
func newMapResolver(uris []string, providers map[string]confmap.Provider, converters []confmap.Converter) (*mapResolver, error) {
	if len(uris) == 0 {
		return nil, errors.New("invalid map resolver config: no URIs")
	}

	if len(providers) == 0 {
		return nil, errors.New("invalid map resolver config: no map providers")
	}

	// Safe copy, ensures the slices and maps cannot be changed from the caller.
	urisCopy := make([]string, len(uris))
	copy(urisCopy, uris)
	providersCopy := make(map[string]confmap.Provider, len(providers))
	for k, v := range providers {
		providersCopy[k] = v
	}
	convertersCopy := make([]confmap.Converter, len(converters))
	copy(convertersCopy, converters)

	return &mapResolver{
		uris:       urisCopy,
		providers:  providersCopy,
		converters: convertersCopy,
		watcher:    make(chan error, 1),
	}, nil
}

// Resolve returns the configuration as a confmap.Conf, or error otherwise.
//
// Should never be called concurrently with itself, Watch or Shutdown.
func (mr *mapResolver) Resolve(ctx context.Context) (*confmap.Conf, error) {
	// First check if already an active watching, close that if any.
	if err := mr.closeIfNeeded(ctx); err != nil {
		return nil, fmt.Errorf("cannot close previous watch: %w", err)
	}

	// Retrieves individual configurations from all URIs in the given order, and merge them in retMap.
	retMap := confmap.New()
	for _, uri := range mr.uris {
		// For backwards compatibility:
		// - empty url scheme means "file".
		// - "^[A-z]:" also means "file"
		scheme := "file"
		if idx := strings.Index(uri, ":"); idx != -1 && !driverLetterRegexp.MatchString(uri) {
			scheme = uri[:idx]
		} else {
			uri = scheme + ":" + uri
		}
		p, ok := mr.providers[scheme]
		if !ok {
			return nil, fmt.Errorf("scheme %q is not supported for uri %q", scheme, uri)
		}
		ret, err := p.Retrieve(ctx, uri, mr.onChange)
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
	for _, confConv := range mr.converters {
		if err := confConv.Convert(ctx, retMap); err != nil {
			return nil, fmt.Errorf("cannot convert the confmap.Conf: %w", err)
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
func (mr *mapResolver) Watch() <-chan error {
	return mr.watcher
}

// Shutdown signals that the provider is no longer in use and the that should close
// and release any resources that it may have created. It terminates the Watch channel.
//
// Should never be called concurrently with itself or Get.
func (mr *mapResolver) Shutdown(ctx context.Context) error {
	close(mr.watcher)

	var errs error
	errs = multierr.Append(errs, mr.closeIfNeeded(ctx))
	for _, p := range mr.providers {
		errs = multierr.Append(errs, p.Shutdown(ctx))
	}

	return errs
}

func (mr *mapResolver) onChange(event *confmap.ChangeEvent) {
	// TODO: Remove check for configsource.ErrSessionClosed when providers updated to not call onChange when closed.
	if !errors.Is(event.Error, configsource.ErrSessionClosed) {
		mr.watcher <- event.Error
	}
}

func (mr *mapResolver) closeIfNeeded(ctx context.Context) error {
	var err error
	for _, ret := range mr.closers {
		err = multierr.Append(err, ret(ctx))
	}
	return err
}

func makeMapProvidersMap(providers ...confmap.Provider) map[string]confmap.Provider {
	ret := make(map[string]confmap.Provider, len(providers))
	for _, provider := range providers {
		ret[provider.Scheme()] = provider
	}
	return ret
}
