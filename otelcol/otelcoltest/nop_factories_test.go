// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
