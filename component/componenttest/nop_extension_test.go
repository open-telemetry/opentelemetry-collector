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

package componenttest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config"
)

func TestNewNopExtensionFactory(t *testing.T) {
	factory := NewNopExtensionFactory()
	require.NotNil(t, factory)
	assert.Equal(t, config.Type("nop"), factory.Type())
	cfg := factory.CreateDefaultConfig()
	assert.Equal(t, &nopExtensionConfig{ExtensionSettings: config.NewExtensionSettings(config.NewID("nop"))}, cfg)

	traces, err := factory.CreateExtension(context.Background(), NewNopExtensionCreateSettings(), cfg)
	require.NoError(t, err)
	assert.NoError(t, traces.Start(context.Background(), NewNopHost()))
	assert.NoError(t, traces.Shutdown(context.Background()))
}
