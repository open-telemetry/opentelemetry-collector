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

	require.Len(t, nopFactories.Receivers, 1)
	nopReceiverFactory, ok := nopFactories.Receivers[nopType]
	require.True(t, ok)
	require.Equal(t, nopType, nopReceiverFactory.Type())

	require.Len(t, nopFactories.Processors, 1)
	nopProcessorFactory, ok := nopFactories.Processors[nopType]
	require.True(t, ok)
	require.Equal(t, nopType, nopProcessorFactory.Type())

	require.Len(t, nopFactories.Exporters, 1)
	nopExporterFactory, ok := nopFactories.Exporters[nopType]
	require.True(t, ok)
	require.Equal(t, nopType, nopExporterFactory.Type())

	require.Len(t, nopFactories.Extensions, 1)
	nopExtensionFactory, ok := nopFactories.Extensions[nopType]
	require.True(t, ok)
	require.Equal(t, nopType, nopExtensionFactory.Type())

	require.Len(t, nopFactories.Connectors, 1)
	nopConnectorFactory, ok := nopFactories.Connectors[nopType]
	require.True(t, ok)
	require.Equal(t, nopType, nopConnectorFactory.Type())
}
