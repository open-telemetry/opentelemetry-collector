// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componenttest

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
)

func TestNewNopSettings(t *testing.T) {
	ns := NewNopSettings()
	require.NotNil(t, ns)
	require.IsType(t, &component.Settings{}, ns)
}
