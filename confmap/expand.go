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
)

// schemePattern defines the regexp pattern for scheme names.
// Scheme name consist of a sequence of characters beginning with a letter and followed by any
// combination of letters, digits, plus ("+"), period ("."), or hyphen ("-").
const schemePattern = `[A-Za-z][A-Za-z0-9+.-]+`

var (
	// Need to match new line as well in the OpaqueValue, so setting the "s" flag. See https://pkg.go.dev/regexp/syntax.
	uriRegexp = regexp.MustCompile(`(?s:^(?P<Scheme>` + schemePattern + `):(?P<OpaqueValue>.*)$)`)

	errTooManyRecursiveExpansions = errors.New("too many recursive expansions")
)

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
		if !strings.Contains(v, "${") || !strings.Contains(v, "}") {
			// No URIs to expand.
			return value, false, nil
		}
		// Embedded or nested URIs.
		return mr.findAndExpandURI(ctx, v)
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

// findURI attempts to find the first expandable URI in input. It returns an expandable
// URI, or an empty string if none are found.
// Note: findURI is only called when input contains a closing bracket.
func findURI(input string) string {
	closeIndex := strings.Index(input, "}")
	remaining := input[closeIndex+1:]
	openIndex := strings.LastIndex(input[:closeIndex+1], "${")

	// if there is a missing "${" or the uri does not contain ":", check the next URI.
	if openIndex < 0 || !strings.Contains(input[openIndex:closeIndex+1], ":") {
		// if remaining does not contain "}", there are no URIs left: stop recursion.
		if !strings.Contains(remaining, "}") {
			return ""
		}
		return findURI(remaining)
	}

	return input[openIndex : closeIndex+1]
}

// findAndExpandURI attempts to find and expand the first occurrence of an expandable URI in input. If an expandable URI is found it
// returns the input with the URI expanded, true and nil. Otherwise, it returns the unchanged input, false and the expanding error.
func (mr *Resolver) findAndExpandURI(ctx context.Context, input string) (any, bool, error) {
	uri := findURI(input)
	if uri == "" {
		// No URI found, return.
		return input, false, nil
	}
	if uri == input {
		// If the value is a single URI, then the return value can be anything.
		// This is the case `foo: ${file:some_extra_config.yml}`.
		return mr.expandURI(ctx, input)
	}
	expanded, changed, err := mr.expandURI(ctx, uri)
	if err != nil {
		return input, false, err
	}
	repl, err := toString(uri, expanded)
	if err != nil {
		return input, false, err
	}
	return strings.ReplaceAll(input, uri, repl), changed, err
}

// toString attempts to convert input to a string.
func toString(strURI string, input any) (string, error) {
	// This list must be kept in sync with checkRawConfType.
	val := reflect.ValueOf(input)
	switch val.Kind() {
	case reflect.String:
		return val.String(), nil
	case reflect.Int, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(val.Int(), 10), nil
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(val.Float(), 'f', -1, 64), nil
	case reflect.Bool:
		return strconv.FormatBool(val.Bool()), nil
	default:
		return "", fmt.Errorf("expanding %v, expected convertable to string value type, got %q(%T)", strURI, input, input)
	}
}

func (mr *Resolver) expandURI(ctx context.Context, uri string) (any, bool, error) {
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
