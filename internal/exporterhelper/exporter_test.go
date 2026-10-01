// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExporterHelper(t *testing.T) {
	require.False(t, NewExporterHelper(false).BatchingEnabled())
	require.True(t, NewExporterHelper(true).BatchingEnabled())
}
