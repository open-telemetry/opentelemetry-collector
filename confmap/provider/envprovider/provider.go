// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:generate mdatagen metadata.yaml

package envprovider // import "go.opentelemetry.io/collector/confmap/provider/envprovider"

import (
	"context"
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/internal/envvar"
)

const (
	schemeName = "env"
)

type provider struct {
	logger *zap.Logger
}

// NewFactory returns a factory for a confmap.Provider that reads the configuration from the given environment variable.
//
// This Provider supports "env" scheme, and can be called with a selector:
// `env:NAME_OF_ENVIRONMENT_VARIABLE`
//
// A default value for unset variable can be provided after :- suffix, for example:
// `env:NAME_OF_ENVIRONMENT_VARIABLE:-default_value`
//
// See also: https://opentelemetry.io/docs/specs/otel/configuration/data-model/#environment-variable-substitution
func NewFactory() confmap.ProviderFactory {
	return confmap.NewProviderFactory(newProvider)
}

func newProvider(ps confmap.ProviderSettings) confmap.Provider {
	return &provider{
		logger: ps.Logger,
	}
}

func (emp *provider) Retrieve(_ context.Context, uri string, _ confmap.WatcherFunc) (*confmap.Retrieved, error) {
	if !strings.HasPrefix(uri, schemeName+":") {
		return nil, fmt.Errorf("%q uri is not supported by %q provider", uri, schemeName)
	}
	envVarName, defaultValuePtr := parseEnvVarURI(uri[len(schemeName)+1:])
	if !envvar.ValidationRegexp.MatchString(envVarName) {
		return nil, fmt.Errorf("environment variable %q has invalid name: must match regex %s", envVarName, envvar.ValidationPattern)
	}

	val, exists := os.LookupEnv(envVarName)
	if !exists {
		if defaultValuePtr != nil {
			val = *defaultValuePtr
		} else {
			emp.logger.Warn("Configuration references unset environment variable", zap.String("name", envVarName))
		}
	} else if val == "" {
		emp.logger.Info("Configuration references empty environment variable", zap.String("name", envVarName))
	}

	return confmap.NewRetrievedFromYAML([]byte(val))
}

func (*provider) Scheme() string {
	return schemeName
}

func (*provider) Shutdown(context.Context) error {
	return nil
}

// returns (var name, default value)
func parseEnvVarURI(uri string) (string, *string) {
	const defaultSuffix = ":-"
	name, defaultValue, hasDefault := strings.Cut(uri, defaultSuffix)
	if hasDefault {
		return name, &defaultValue
	}
	return uri, nil
}
