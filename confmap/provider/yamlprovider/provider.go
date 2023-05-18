// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package yamlprovider // import "go.opentelemetry.io/collector/confmap/provider/yamlprovider"

import (
	"context"
	"fmt"
	"strings"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/internal"
)

const schemeName = "yaml"

type provider struct{}

// New returns a new confmap.Provider that allows to provide yaml bytes.
//
// This Provider supports "yaml" scheme, and can be called with a "uri" that follows:
//
//	bytes-uri = "yaml:" yaml-bytes
//
// Examples:
// `yaml:processors::batch::timeout: 2s`
// `yaml:processors::batch/foo::timeout: 3s`
func New() confmap.Provider {
	return &provider{}
}

func (s *provider) Retrieve(_ context.Context, uri string, _ confmap.WatcherFunc) (*confmap.Retrieved, error) {
	if !strings.HasPrefix(uri, schemeName+":") {
		return nil, fmt.Errorf("%q uri is not supported by %q provider", uri, schemeName)
	}

	return internal.NewRetrievedFromYAML([]byte(uri[len(schemeName)+1:]))
}

func (*provider) Scheme() string {
	return schemeName
}

func (s *provider) Shutdown(context.Context) error {
	return nil
}
