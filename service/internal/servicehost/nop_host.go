// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package servicehost // import "go.opentelemetry.io/collector/service/internal/servicehost"

import (
	"go.opentelemetry.io/collector/component"
)

// nopHost mocks a receiver.ReceiverHost for test purposes.
type nopHost struct{}

func (n nopHost) ReportFatalError(_ error) {
}

func (n nopHost) ReportComponentStatus(_ *component.InstanceID, _ *component.StatusEvent) {
}

func (n nopHost) GetFactory(_ component.Kind, _ component.Type) component.Factory {
	return nil
}

func (n nopHost) GetExtensions() map[component.ID]component.Component {
	return nil
}

func (n nopHost) GetExporters() map[component.DataType]map[component.ID]component.Component {
	return nil
}

// NewNopHost returns a new instance of nopHost with proper defaults for most tests.
func NewNopHost() Host {
	return &nopHost{}
}
