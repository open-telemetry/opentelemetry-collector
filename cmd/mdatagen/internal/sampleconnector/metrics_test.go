// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package sampleconnector

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/cmd/mdatagen/internal/sampleconnector/internal/metadata"
	"go.opentelemetry.io/collector/connector/connectortest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

// TestGeneratedMetrics verifies that the internal/metadata API is generated correctly.
func TestGeneratedMetrics(t *testing.T) {
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), connectortest.NewNopSettings(metadata.Type))
	m := mb.Emit()
	require.Equal(t, 0, m.ResourceMetrics().Len())
}

func TestNopConnector(t *testing.T) {
	connector, err := createMetricsToMetricsConnector(context.Background(), connectortest.NewNopSettings(metadata.Type), newMdatagenNopHost(), new(consumertest.MetricsSink))
	require.NoError(t, err)
	require.False(t, connector.Capabilities().MutatesData)
	require.NoError(t, connector.ConsumeMetrics(context.Background(), pmetric.NewMetrics()))
}
