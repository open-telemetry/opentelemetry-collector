// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package metadata

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric"
	embeddedmetric "go.opentelemetry.io/otel/metric/embedded"

	"go.opentelemetry.io/collector/component/componenttest"
)

type mockRegistration struct {
	embeddedmetric.Registration
	unregistered bool
}

func (r *mockRegistration) Unregister() error {
	r.unregistered = true
	return nil
}

func TestTelemetryBuilderShutdown(t *testing.T) {
	tb, err := NewTelemetryBuilder(componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)

	reg := &mockRegistration{}
	tb.registrations = []metric.Registration{reg}

	tb.Shutdown()

	require.True(t, reg.unregistered)
}
