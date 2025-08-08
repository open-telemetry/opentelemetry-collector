// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componenttest // import "go.opentelemetry.io/collector/component/componenttest"

import (
	"go.opentelemetry.io/collector/component"
)

// nopHost mocks a [component.Host] for testing purposes.
type nopHost struct{}

// NewNopHost returns a [component.Host] that returns empty values
// from method calls. This host is intended to be used in tests
// where a bare-minimum host is desired.
func NewNopHost() component.Host {
	return &nopHost{}
}

func (nh *nopHost) GetFactory(component.Kind, component.Type) component.Factory {
	return nil
}

// GetExtensions returns a `nil` extensions map.
func (nh *nopHost) GetExtensions() map[component.ID]component.Component {
	return nil
}
