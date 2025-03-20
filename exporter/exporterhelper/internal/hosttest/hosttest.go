// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package hosttest // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/hosttest"

import (
	"go.opentelemetry.io/collector/component"
)

var _ component.Host = (*MockHost)(nil)

type MockHost struct {
	component.Host
	Ext map[component.ID]component.Component
}

func (nh *MockHost) GetExtensions() map[component.ID]component.Component {
	return nh.Ext
}
