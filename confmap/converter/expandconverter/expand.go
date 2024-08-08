// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package expandconverter // import "go.opentelemetry.io/collector/confmap/converter/expandconverter"

import (
	"context"
	"fmt"
	"os"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/internal/envvar"
)

type converter struct {
	logger *zap.Logger

	// Record of which env vars we have logged a warning for
	loggedDeprecations map[string]struct{}
}

// NewFactory returns a factory for a  confmap.Converter,
// which expands all environment variables for a given confmap.Conf.
//
// Deprecated: [v0.107.0] BASH-style env var expansion is deprecated. Use the `envprovider` instead to expand `${FOO}` and `${env:FOO}`.
// Using the expandconverter with `confmap.Resolver` will cause double escaping, so `$$$$` -> `$` instead of `$$`.
func NewFactory() confmap.ConverterFactory {
	return confmap.NewConverterFactory(newConverter)
}

func newConverter(set confmap.ConverterSettings) confmap.Converter {
	return converter{
		loggedDeprecations: make(map[string]struct{}),
		logger:             set.Logger,
	}
}

// Deprecated: [v0.107.0] BASH-style env var expansion is deprecated. Use the `envprovider` instead to expand `${FOO}` and `${env:FOO}`.
// Using the expandconverter with `confmap.Resolver` will cause double escaping, so `$$$$` -> `$` instead of `$$`.
func (c converter) Convert(_ context.Context, conf *confmap.Conf) error {
	var err error
	out := make(map[string]any)
	for _, k := range conf.AllKeys() {
		out[k], err = c.expandStringValues(conf.Get(k))
		if err != nil {
			return err
		}
	}
	return conf.Merge(confmap.NewFromStringMap(out))
}

func (c converter) expandStringValues(value any) (any, error) {
	var err error
	switch v := value.(type) {
	case string:
		return c.expandEnv(v)
	case []any:
		nslice := make([]any, 0, len(v))
		for _, vint := range v {
			var nv any
			nv, err = c.expandStringValues(vint)
			if err != nil {
				return nil, err
			}
			nslice = append(nslice, nv)
		}
		return nslice, nil
	case map[string]any:
		nmap := map[string]any{}
		for mk, mv := range v {
			nmap[mk], err = c.expandStringValues(mv)
			if err != nil {
				return nil, err
			}
		}
		return nmap, nil
	default:
		return v, nil
	}
}

func (c converter) expandEnv(s string) (string, error) {
	var err error
	res := os.Expand(s, func(str string) string {
		// This allows escaping environment variable substitution via $$, e.g.
		// - $FOO will be substituted with env var FOO
		// - $$FOO will be replaced with $FOO
		// - $$$FOO will be replaced with $ + substituted env var FOO
		// TODO: Move the escaping of $$ out from the expand converter to the resolver.
		if str == "$" {
			return "$"
		}

		// For $ENV style environment variables os.Expand returns once it hits a character that isn't an underscore or
		// an alphanumeric character - so we cannot detect those malformed environment variables.
		// For ${ENV} style variables we can detect those kinds of env var names!
		if !envvar.ValidationRegexp.MatchString(str) {
			err = fmt.Errorf("environment variable %q has invalid name: must match regex %s", str, envvar.ValidationPattern)
			return ""
		}
		val, exists := os.LookupEnv(str)
		if !exists {
			c.logger.Warn("Configuration references unset environment variable", zap.String("name", str))
		} else if len(val) == 0 {
			c.logger.Info("Configuration references empty environment variable", zap.String("name", str))
		}
		return val
	})
	return res, err
}
