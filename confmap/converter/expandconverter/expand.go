// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package expandconverter // import "go.opentelemetry.io/collector/confmap/converter/expandconverter"

import (
	"context"
	"os"

	"go.opentelemetry.io/collector/confmap"
)

type converter struct{}

// New returns a confmap.Converter, that expands all environment variables for a given confmap.Conf.
//
// Notice: This API is experimental.
func New() confmap.Converter {
	return converter{}
}

func (converter) Convert(_ context.Context, conf *confmap.Conf) error {
	out := make(map[string]any)
	for _, k := range conf.AllKeys() {
		out[k] = expandStringValues(conf.Get(k))
	}
	return conf.Merge(confmap.NewFromStringMap(out))
}

func expandStringValues(value any) any {
	switch v := value.(type) {
	case string:
		return expandEnv(v)
	case []any:
		nslice := make([]any, 0, len(v))
		for _, vint := range v {
			nslice = append(nslice, expandStringValues(vint))
		}
		return nslice
	case map[string]any:
		nmap := map[string]any{}
		for mk, mv := range v {
			nmap[mk] = expandStringValues(mv)
		}
		return nmap
	default:
		return v
	}
}

func expandEnv(s string) string {
	return os.Expand(s, func(str string) string {
		// This allows escaping environment variable substitution via $$, e.g.
		// - $FOO will be substituted with env var FOO
		// - $$FOO will be replaced with $FOO
		// - $$$FOO will be replaced with $ + substituted env var FOO
		if str == "$" {
			return "$"
		}
		return os.Getenv(str)
	})
}
