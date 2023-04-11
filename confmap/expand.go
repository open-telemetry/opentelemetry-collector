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

	errURILimit = errors.New("reached limit of 100 URIs")
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

		if strings.Count(v, "}") > 100 {
			// Too many closing brackets. Don't expand to protect from too deep recursion.
			return "", false, errURILimit
		}

		uri := findURI(v)
		if uri != "" && uri == value {
			// If the value is a single URI, then the return value can be anything.
			// This is the case `foo: ${file:some_extra_config.yml}`.
			return mr.expandURI(ctx, v)
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

// findURI finds the URI corresponding to the first closing bracket in input. It returns
// the URI if it is expandable, or an empty string if it is not expandable.
// Note: findURI is only called when input contains a closing bracket.
func findURI(input string) string {
	closeIndex := strings.Index(input, "}")
	openIndex := strings.LastIndex(input[:closeIndex+1], "${")
	if openIndex < 0 {
		// Should not expand because there is a missing ${.
		return ""
	}

	uri := input[openIndex : closeIndex+1]

	if !strings.Contains(uri, ":") {
		// Should not expand. This is expanded in the expandconverter.
		return ""
	}

	return uri
}

// findAndExpandURI attempts to find and expand the first occurrence of an expandable URI in input.
func (mr *Resolver) findAndExpandURI(ctx context.Context, input string) (output string, changed bool, err error) {
	var repl string
	var expandedRemaining string

	uri := findURI(input)
	if uri == "" {
		// The first URI in input is not expandable. Strip the first URI from input and check if
		// other URIs are expandable.
		closeIndex := strings.Index(input, "}")
		noExpand := input[:closeIndex+1]
		remaining := input[closeIndex+1:]

		// if remaining does not contain }, there are no URIs left: stop recursion.
		if !strings.Contains(remaining, "}") {
			return input, changed, err
		}

		expandedRemaining, changed, err = mr.findAndExpandURI(ctx, remaining)
		return noExpand + expandedRemaining, changed, err
	}
	repl, changed, err = mr.expandStringURI(ctx, uri)
	input = strings.ReplaceAll(input, uri, repl)
	return input, changed, err
}

// expandStringURI tries to expand uri as a string. If an expandable URI is found and it can be converted to a string,
// it returns the expanded uris string value, true and nil. Otherwise, it returns the unchanged input, false and the expanding error.
func (mr *Resolver) expandStringURI(ctx context.Context, uri string) (string, bool, error) {
	expanded, changed, err := mr.expandURI(ctx, uri)
	if err != nil {
		return uri, changed, err
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
		return uri, changed, fmt.Errorf("expanding %v, expected string value type, got %T", uri, expanded)
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
