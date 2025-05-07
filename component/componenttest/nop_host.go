// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componenttest // import "go.opentelemetry.io/collector/component/componenttest"

import (
	"go.opentelemetry.io/collector/component"
)

// nopHost mocks a receiver.ReceiverHost for test purposes.
type nopHost struct{}

// NewNopHost returns a new instance of nopHost with proper defaults for most tests.
func NewNopHost() component.Host {
	return &nopHost{}
}

func (nh *nopHost) GetFactory(component.Kind, component.Type) component.Factory {
	return nil
}

func (nh *nopHost) GetExtensions() map[component.ID]component.Component {
	return nil
}
