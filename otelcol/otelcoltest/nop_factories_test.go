// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcoltest

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
)

func TestNopFactories(t *testing.T) {
	nopFactories, err := NopFactories()
	require.NoError(t, err)

	require.Equal(t, 1, len(nopFactories.Receivers))
	nopReceiverFactory, ok := nopFactories.Receivers["nop"]
	require.True(t, ok)
	require.Equal(t, component.Type("nop"), nopReceiverFactory.Type())

	require.Equal(t, 1, len(nopFactories.Processors))
	nopProcessorFactory, ok := nopFactories.Processors["nop"]
	require.True(t, ok)
	require.Equal(t, component.Type("nop"), nopProcessorFactory.Type())

	require.Equal(t, 1, len(nopFactories.Exporters))
	nopExporterFactory, ok := nopFactories.Exporters["nop"]
	require.True(t, ok)
	require.Equal(t, component.Type("nop"), nopExporterFactory.Type())

	require.Equal(t, 1, len(nopFactories.Extensions))
	nopExtensionFactory, ok := nopFactories.Extensions["nop"]
	require.True(t, ok)
	require.Equal(t, component.Type("nop"), nopExtensionFactory.Type())

	require.Equal(t, 1, len(nopFactories.Connectors))
	nopConnectorFactory, ok := nopFactories.Connectors["nop"]
	require.True(t, ok)
	require.Equal(t, component.Type("nop"), nopConnectorFactory.Type())
}
