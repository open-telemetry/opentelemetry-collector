// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package expandconverter // import "go.opentelemetry.io/collector/confmap/converter/expandconverter"

import (
	"context"
	"fmt"
	"os"
	"regexp"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/confmap"
)

type converter struct {
	logger *zap.Logger

	// Record of which env vars we have logged a warning for
	loggedDeprecations map[string]struct{}
}

// New returns a confmap.Converter, that expands all environment variables for a given confmap.Conf.
//
// Notice: This API is experimental.
//
// Deprecated: [v0.99.0] Use NewFactory instead.
func New(_ confmap.ConverterSettings) confmap.Converter {
	return converter{
		loggedDeprecations: make(map[string]struct{}),
		logger:             zap.NewNop(), // TODO: pass logger in ConverterSettings
	}
}

// NewFactory returns a factory for a  confmap.Converter,
// which expands all environment variables for a given confmap.Conf.
func NewFactory() confmap.ConverterFactory {
	return confmap.NewConverterFactory(New)
}

func (c converter) Convert(_ context.Context, conf *confmap.Conf) error {
	out := make(map[string]any)
	for _, k := range conf.AllKeys() {
		out[k] = c.expandStringValues(conf.Get(k))
	}
	return conf.Merge(confmap.NewFromStringMap(out))
}

func (c converter) expandStringValues(value any) any {
	switch v := value.(type) {
	case string:
		return c.expandEnv(v)
	case []any:
		nslice := make([]any, 0, len(v))
		for _, vint := range v {
			nslice = append(nslice, c.expandStringValues(vint))
		}
		return nslice
	case map[string]any:
		nmap := map[string]any{}
		for mk, mv := range v {
			nmap[mk] = c.expandStringValues(mv)
		}
		return nmap
	default:
		return v
	}
}

func (c converter) expandEnv(s string) string {
	return os.Expand(s, func(str string) string {
		// Matches on $VAR style environment variables
		// in order to make sure we don't log a warning for ${VAR}
		var regex = regexp.MustCompile(fmt.Sprintf(`\$%s`, regexp.QuoteMeta(str)))
		if _, exists := c.loggedDeprecations[str]; !exists && regex.MatchString(s) {
			msg := fmt.Sprintf("Variable substitution using $VAR will be deprecated in favor of ${VAR} and ${env:VAR}, please update $%s", str)
			c.logger.Warn(msg, zap.String("variable", str))
			c.loggedDeprecations[str] = struct{}{}
		}
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
