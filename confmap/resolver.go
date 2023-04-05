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
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/featuregate"
)

// schemePattern defines the regexp pattern for scheme names.
// Scheme name consist of a sequence of characters beginning with a letter and followed by any
// combination of letters, digits, plus ("+"), period ("."), or hyphen ("-").
const schemePattern = `[A-Za-z][A-Za-z0-9+.-]+`

var (
	// follows drive-letter specification:
	// https://datatracker.ietf.org/doc/html/draft-kerwin-file-scheme-07.html#section-2.2
	driverLetterRegexp = regexp.MustCompile("^[A-z]:")

	// Need to match new line as well in the OpaqueValue, so setting the "s" flag. See https://pkg.go.dev/regexp/syntax.
	uriRegexp = regexp.MustCompile(`(?s:^(?P<Scheme>` + schemePattern + `):(?P<OpaqueValue>.*)$)`)

	errTooManyRecursiveExpansions = errors.New("too many recursive expansions")
)

var _ = featuregate.GlobalRegistry().MustRegister(
	"confmap.expandEnabled",
	featuregate.StageStable,
	featuregate.WithRegisterRemovalVersion("v0.75.0"),
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

func (mr *Resolver) expandValueRecursively(ctx context.Context, value any) (any, error) {
	for i := 0; i < 100; i++ {
		val, changed, err := mr.expandValue(ctx, value)
		if err != nil {
			return nil, err
		}
		if !changed {
			return val, nil
		}
		value = val
	}
	return nil, errTooManyRecursiveExpansions
}

func (mr *Resolver) expandValue(ctx context.Context, value any) (any, bool, error) {
	switch v := value.(type) {
	case string:
		// No URIs to expand.
		if !strings.Contains(v, "${") || !strings.Contains(v, "}") {
			return value, false, nil
		}

		// Don't expand. Too many closing brackets.
		if strings.Count(v, "}") > 100 {
			return value, false, nil
		}

		URI, expand := mr.findURI(v)
		// If the value is a single URI, then the return value can be anything.
		// This is the case `foo: ${file:some_extra_config.yml}`.
		if expand && URI == value {
			return mr.expandStringURI(ctx, v)
		}

		// Embedded or nested URIs.
		return mr.expandURIs(ctx, v)
	case []any:
		nslice := make([]any, 0, len(v))
		nchanged := false
		for _, vint := range v {
			val, changed, err := mr.expandValue(ctx, vint)
			if err != nil {
				return nil, false, err
			}
			nslice = append(nslice, val)
			nchanged = nchanged || changed
		}
		return nslice, nchanged, nil
	case map[string]any:
		nmap := map[string]any{}
		nchanged := false
		for mk, mv := range v {
			val, changed, err := mr.expandValue(ctx, mv)
			if err != nil {
				return nil, false, err
			}
			nmap[mk] = val
			nchanged = nchanged || changed
		}
		return nmap, nchanged, nil
	}
	return value, false, nil
}

// findURI will find the URI corresponding to the first closing bracket in input.
// findURI is only called when input contains a }.
func (mr *Resolver) findURI(input string) (string, bool) {
	closeIndex := closeIndex(input)
	openIndex := openIndex(input[:closeIndex+1])
	// Should not expand because there is a missing ${.
	if openIndex < 0 {
		return "", false
	}

	URI := input[openIndex : closeIndex+1]

	// Should not expand. This is expanded in the expandconverter.
	if !strings.Contains(URI, ":") {
		return "", false
	}

	return URI, true
}

func closeIndex(s string) int {
	return strings.Index(s, "}")
}

func openIndex(s string) int {
	return strings.LastIndex(s, "${")
}

func (mr *Resolver) expandURIs(ctx context.Context, input string) (string, bool, error) {
	var err error
	var changed bool

	URI, expand := mr.findURI(input)
	if expand {
		input, changed, err = mr.expandableURI(ctx, input, URI)
	}
	// Check if other URIs are expandable.
	if !expand {
		input, changed, err = mr.nonExpandableURI(ctx, input)
	}
	return input, changed, err
}

func (mr *Resolver) expandableURI(ctx context.Context, input string, uri string) (string, bool, error) {
	repl, changed, err := mr.expandURI(ctx, uri)
	return strings.ReplaceAll(input, uri, repl), changed, err
}

// nonExpandableURI finds the next expandable URI in input and expands it.
func (mr *Resolver) nonExpandableURI(ctx context.Context, input string) (string, bool, error) {
	var err error
	var expandedRemaining string
	var changed bool

	noExpand := input[:closeIndex(input)+1]
	remaining := input[closeIndex(input)+1:]

	if strings.Contains(remaining, "}") {
		expandedRemaining, changed, err = mr.expandURIs(ctx, remaining)
		return noExpand + expandedRemaining, changed, err
	}
	return input, changed, err
}

func (mr *Resolver) expandURI(ctx context.Context, value string) (string, bool, error) {
	expanded, changed, err := mr.expandStringURI(ctx, value)
	if err != nil {
		return "", changed, err
	}
	val := reflect.ValueOf(expanded)
	switch val.Kind() {
	case reflect.String:
		return val.String(), changed, err
	case reflect.Int, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(val.Int(), 10), changed, err
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(val.Float(), 'f', -1, 64), changed, err
	case reflect.Bool:
		return strconv.FormatBool(val.Bool()), changed, err
	default:
		return value, changed, fmt.Errorf("expanding %v, expected string value type, got %T", value, expanded)
	}
}

func (mr *Resolver) expandStringURI(ctx context.Context, uri string) (any, bool, error) {
	lURI, err := newLocation(uri[2 : len(uri)-1])
	if err != nil {
		return nil, false, err
	}
	if strings.Contains(lURI.opaqueValue, "$") {
		return nil, false, fmt.Errorf("the uri %q contains unsupported characters ('$')", lURI.asString())
	}
	ret, err := mr.retrieveValue(ctx, lURI)
	if err != nil {
		return nil, false, err
	}
	mr.closers = append(mr.closers, ret.Close)
	val, err := ret.AsRaw()
	return val, true, err
}

type location struct {
	scheme      string
	opaqueValue string
}

func (c location) asString() string {
	return c.scheme + ":" + c.opaqueValue
}

func newLocation(uri string) (location, error) {
	submatches := uriRegexp.FindStringSubmatch(uri)
	if len(submatches) != 3 {
		return location{}, fmt.Errorf("invalid uri: %q", uri)
	}
	return location{scheme: submatches[1], opaqueValue: submatches[2]}, nil
}

func (mr *Resolver) retrieveValue(ctx context.Context, uri location) (*Retrieved, error) {
	p, ok := mr.providers[uri.scheme]
	if !ok {
		return nil, fmt.Errorf("scheme %q is not supported for uri %q", uri.scheme, uri.asString())
	}
	return p.Retrieve(ctx, uri.asString(), mr.onChange)
}
