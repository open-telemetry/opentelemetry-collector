// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package hosttest

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
)

// nopExtension acts as an extension for testing purposes.
type nopExtension struct {
	component.StartFunc
	component.ShutdownFunc
}

func TestMockHost(t *testing.T) {
	componenttest.NewNopHost()
	ext := map[component.ID]component.Component{
		component.MustNewID("test"): &nopExtension{},
	}
	host := NewHost(ext)
	assert.Equal(t, ext, host.GetExtensions())
}
