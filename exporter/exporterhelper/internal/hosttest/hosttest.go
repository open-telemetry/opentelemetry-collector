// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package hosttest // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/hosttest"

import (
	"go.opentelemetry.io/collector/component"
)

func NewHost(ext map[component.ID]component.Component) component.Host {
	return &mockHost{ext: ext}
}

type mockHost struct {
	ext map[component.ID]component.Component
}

func (nh *mockHost) GetExtensions() map[component.ID]component.Component {
	return nh.ext
}
