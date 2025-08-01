// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package yamlprovider // import "go.opentelemetry.io/collector/confmap/provider/yamlprovider"

//go:generate mdatagen metadata.yaml

import (
	"context"
	"fmt"
	"strings"

	"go.opentelemetry.io/collector/confmap"
)

const schemeName = "yaml"

type provider struct{}

// NewFactory returns a factory for a confmap.Provider that allows to provide yaml bytes.
//
// This Provider supports "yaml" scheme, and can be called with a "uri" that follows:
//
//	bytes-uri = "yaml:" yaml-bytes
//
// Examples:
// `yaml:exporters::otlphttp::sending_queue::batch::flush_timeout: 2s`
// `yaml:exporters::otlphttp/foo::sending_queue::batch::flush_timeout: 2s`
func NewFactory() confmap.ProviderFactory {
	return confmap.NewProviderFactory(newProvider)
}

func newProvider(confmap.ProviderSettings) confmap.Provider {
	return &provider{}
}

func (s *provider) Retrieve(_ context.Context, uri string, _ confmap.WatcherFunc) (*confmap.Retrieved, error) {
	if !strings.HasPrefix(uri, schemeName+":") {
		return nil, fmt.Errorf("%q uri is not supported by %q provider", uri, schemeName)
	}

	return confmap.NewRetrievedFromYAML([]byte(uri[len(schemeName)+1:]))
}

func (*provider) Scheme() string {
	return schemeName
}

func (s *provider) Shutdown(context.Context) error {
	return nil
}
