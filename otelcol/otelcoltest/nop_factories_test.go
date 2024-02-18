// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcoltest

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
)

var nopType = component.MustNewType("nop")

func TestNopFactories(t *testing.T) {
	nopFactories, err := NopFactories()
	require.NoError(t, err)

	require.Equal(t, 1, len(nopFactories.Receivers))
	nopReceiverFactory, ok := nopFactories.Receivers[nopType]
	require.True(t, ok)
	require.Equal(t, nopType, nopReceiverFactory.Type())

	require.Equal(t, 1, len(nopFactories.Processors))
	nopProcessorFactory, ok := nopFactories.Processors[nopType]
	require.True(t, ok)
	require.Equal(t, nopType, nopProcessorFactory.Type())

	require.Equal(t, 1, len(nopFactories.Exporters))
	nopExporterFactory, ok := nopFactories.Exporters[nopType]
	require.True(t, ok)
	require.Equal(t, nopType, nopExporterFactory.Type())

	require.Equal(t, 1, len(nopFactories.Extensions))
	nopExtensionFactory, ok := nopFactories.Extensions[nopType]
	require.True(t, ok)
	require.Equal(t, nopType, nopExtensionFactory.Type())

	require.Equal(t, 1, len(nopFactories.Connectors))
	nopConnectorFactory, ok := nopFactories.Connectors[nopType]
	require.True(t, ok)
	require.Equal(t, nopType, nopConnectorFactory.Type())
}
