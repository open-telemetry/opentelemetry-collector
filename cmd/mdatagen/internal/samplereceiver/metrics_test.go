// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package samplereceiver

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/cmd/mdatagen/internal/samplereceiver/internal/metadata"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

// TestGeneratedMetrics verifies that the internal/metadata API is generated correctly.
func TestGeneratedMetrics(t *testing.T) {
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), receivertest.NewNopCreateSettings())
	m := mb.Emit()
	require.Equal(t, 0, m.ResourceMetrics().Len())
}
