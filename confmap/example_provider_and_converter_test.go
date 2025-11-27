// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confmap_test

import (
	"context"
	"fmt"
	"strings"

	"go.opentelemetry.io/collector/confmap"
)

// mockFileProvider simulates the standard file provider behavior.
// In typical usage, it reads a single file as the root configuration.
// This mock implementation always returns a fixed configuration:
// { "my-config": "${expand:to-expand}" }
type mockFileProvider struct{}

func (d mockFileProvider) Retrieve(_ context.Context, uri string, _ confmap.WatcherFunc) (*confmap.Retrieved, error) {
	expectedURI := "file:mock-file"
	if uri != expectedURI {
		panic("should not happen, the uri is expected to be " + expectedURI + " for mockFileProvider")
	}
	return confmap.NewRetrieved(map[string]any{
		"my-config": "${expand:to-expand}",
	})
}

func (d mockFileProvider) Scheme() string {
	return "file"
}

func (d mockFileProvider) Shutdown(_ context.Context) error {
	return nil
}

// mockExpandProvider simulates a typical inline expansion provider.
// In configurations, you can use expressions like ${SCHEMA:VALUE},
// where the provider associated with SCHEMA is responsible for resolving the value.
type mockExpandProvider struct{}

func (m mockExpandProvider) Retrieve(_ context.Context, uri string, _ confmap.WatcherFunc) (*confmap.Retrieved, error) {
	expectedURI := "expand:to-expand"
	if uri != expectedURI {
		panic("should not happen, the uri is expected to be " + expectedURI + " for mockExpandProvider")
	}
	return confmap.NewRetrieved("expanded")
}

func (m mockExpandProvider) Scheme() string {
	return "expand"
}

func (m mockExpandProvider) Shutdown(_ context.Context) error {
	return nil
}

// mockUpperCaseConverter transforms the value of the `my-config` field in the configuration to uppercase.
type mockUpperCaseConverter struct{}

func (m mockUpperCaseConverter) Convert(_ context.Context, conf *confmap.Conf) error {
	currentValue := conf.Get("my-config")
	expectedValue := "expanded"
	if currentValue != expectedValue {
		panic("should not happen, the value for converter should always be " + expectedValue + " for mockUpperCaseConverter")
	}
	upperCaseConf := confmap.NewFromStringMap(map[string]any{
		"my-config": strings.ToUpper(currentValue.(string)),
	})
	if conf.Merge(upperCaseConf) != nil {
		panic("merge failed, this should not happen in this example.")
	}
	return nil
}

func Example() {
	resolver, err := confmap.NewResolver(confmap.ResolverSettings{
		URIs: []string{"file:mock-file"},
		ProviderFactories: []confmap.ProviderFactory{
			confmap.NewProviderFactory(func(_ confmap.ProviderSettings) confmap.Provider {
				return &mockFileProvider{}
			}),
			confmap.NewProviderFactory(func(_ confmap.ProviderSettings) confmap.Provider {
				return &mockExpandProvider{}
			}),
		},
		ConverterFactories: []confmap.ConverterFactory{
			confmap.NewConverterFactory(func(_ confmap.ConverterSettings) confmap.Converter {
				return &mockUpperCaseConverter{}
			}),
		},
	})
	if err != nil {
		panic(err)
	}

	conf, err := resolver.Resolve(context.Background())
	if err != nil {
		panic(err)
	}
	fmt.Printf("Configuration contains the following:\nmy-config: %s", conf.Get("my-config"))
	// Output: Configuration contains the following:
	// my-config: EXPANDED
}
