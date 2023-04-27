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

package confmap // import "go.opentelemetry.io/collector/confmap"

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/featuregate"
)

// follows drive-letter specification:
// https://datatracker.ietf.org/doc/html/draft-kerwin-file-scheme-07.html#section-2.2
var driverLetterRegexp = regexp.MustCompile("^[A-z]:")

var _ = featuregate.GlobalRegistry().MustRegister(
	"confmap.expandEnabled",
	featuregate.StageStable,
	featuregate.WithRegisterToVersion("v0.75.0"),
	featuregate.WithRegisterDescription("controls whether expanding embedded external config providers URIs"))

// Resolver resolves a configuration as a Conf.
type Resolver struct {
	uris       []location
	providers  map[string]Provider
	converters []Converter

	closers []CloseFunc
	watcher chan error
}

// ResolverSettings are the settings to configure the behavior of the Resolver.
type ResolverSettings struct {
	// URIs locations from where the Conf is retrieved, and merged in the given order.
	// It is required to have at least one location.
	URIs []string

	// Providers is a map of pairs <scheme, Provider>.
	// It is required to have at least one Provider.
	Providers map[string]Provider

	// MapConverters is a slice of Converter.
	Converters []Converter
}

// NewResolver returns a new Resolver that resolves configuration from multiple URIs.
//
// To resolve a configuration the following steps will happen:
//  1. Retrieves individual configurations from all given "URIs", and merge them in the retrieve order.
//  2. Once the Conf is merged, apply the converters in the given order.
//
// After the configuration was resolved the `Resolver` can be used as a single point to watch for updates in
// the configuration data retrieved via the config providers used to process the "initial" configuration and to generate
// the "effective" one. The typical usage is the following:
//
//	Resolver.Resolve(ctx)
//	Resolver.Watch() // wait for an event.
//	Resolver.Resolve(ctx)
//	Resolver.Watch() // wait for an event.
//	// repeat Resolve/Watch cycle until it is time to shut down the Collector process.
//	Resolver.Shutdown(ctx)
//
// `uri` must follow the "<scheme>:<opaque_data>" format. This format is compatible with the URI definition
// (see https://datatracker.ietf.org/doc/html/rfc3986). An empty "<scheme>" defaults to "file" schema.
func NewResolver(set ResolverSettings) (*Resolver, error) {
	if len(set.URIs) == 0 {
		return nil, errors.New("invalid map resolver config: no URIs")
	}

	if len(set.Providers) == 0 {
		return nil, errors.New("invalid map resolver config: no Providers")
	}

	// Safe copy, ensures the slices and maps cannot be changed from the caller.
	uris := make([]location, len(set.URIs))
	for i, uri := range set.URIs {
		// For backwards compatibility:
		// - empty url scheme means "file".
		// - "^[A-z]:" also means "file"
		if driverLetterRegexp.MatchString(uri) || !strings.Contains(uri, ":") {
			uris[i] = location{scheme: "file", opaqueValue: uri}
			continue
		}
		lURI, err := newLocation(uri)
		if err != nil {
			return nil, err
		}
		if _, ok := set.Providers[lURI.scheme]; !ok {
			return nil, fmt.Errorf("unsupported scheme on URI %q", uri)
		}
		uris[i] = lURI
	}
	providersCopy := make(map[string]Provider, len(set.Providers))
	for k, v := range set.Providers {
		providersCopy[k] = v
	}
	convertersCopy := make([]Converter, len(set.Converters))
	copy(convertersCopy, set.Converters)

	return &Resolver{
		uris:       uris,
		providers:  providersCopy,
		converters: convertersCopy,
		watcher:    make(chan error, 1),
	}, nil
}

// Resolve returns the configuration as a Conf, or error otherwise.
//
// Should never be called concurrently with itself, Watch or Shutdown.
func (mr *Resolver) Resolve(ctx context.Context) (*Conf, error) {
	// First check if already an active watching, close that if any.
	if err := mr.closeIfNeeded(ctx); err != nil {
		return nil, fmt.Errorf("cannot close previous watch: %w", err)
	}

	// Retrieves individual configurations from all URIs in the given order, and merge them in retMap.
	retMap := New()
	for _, uri := range mr.uris {
		ret, err := mr.retrieveValue(ctx, uri)
		if err != nil {
			return nil, fmt.Errorf("cannot retrieve the configuration: %w", err)
		}
		mr.closers = append(mr.closers, ret.Close)
		retCfgMap, err := ret.AsConf()
		if err != nil {
			return nil, err
		}
		if err = retMap.Merge(retCfgMap); err != nil {
			return nil, err
		}
	}

	cfgMap := make(map[string]any)
	for _, k := range retMap.AllKeys() {
		val, err := mr.expandValueRecursively(ctx, retMap.Get(k))
		if err != nil {
			return nil, err
		}
		cfgMap[k] = val
	}
	retMap = NewFromStringMap(cfgMap)

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
func (mr *Resolver) Watch() <-chan error {
	return mr.watcher
}

// Shutdown signals that the provider is no longer in use and the that should close
// and release any resources that it may have created. It terminates the Watch channel.
//
// Should never be called concurrently with itself or Get.
func (mr *Resolver) Shutdown(ctx context.Context) error {
	close(mr.watcher)

	var errs error
	errs = multierr.Append(errs, mr.closeIfNeeded(ctx))
	for _, p := range mr.providers {
		errs = multierr.Append(errs, p.Shutdown(ctx))
	}

	return errs
}

func (mr *Resolver) onChange(event *ChangeEvent) {
	mr.watcher <- event.Error
}

func (mr *Resolver) closeIfNeeded(ctx context.Context) error {
	var err error
	for _, ret := range mr.closers {
		err = multierr.Append(err, ret(ctx))
	}
	mr.closers = nil
	return err
}

func (mr *Resolver) retrieveValue(ctx context.Context, uri location) (*Retrieved, error) {
	p, ok := mr.providers[uri.scheme]
	if !ok {
		return nil, fmt.Errorf("scheme %q is not supported for uri %q", uri.scheme, uri.asString())
	}
	return p.Retrieve(ctx, uri.asString(), mr.onChange)
}
