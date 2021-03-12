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
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
)

// GetExtensionConfigFn is used customize the configuration passed to the verification.
// This is used to change ports or provide values required but not provided by the
// default configuration.
type GetExtensionConfigFn func() configmodels.Extension

// VerityExtensionLifecycle is used to test if an extension type can handle the typical
// lifecycle of a component. The getConfigFn parameter only need to be specified if
// the test can't be done with the default configuration for the component.
func VerityExtensionLifecycle(t *testing.T, factory component.ExtensionFactory, getConfigFn GetExtensionConfigFn) {
	ctx := context.Background()
	host := NewAssertNoError(t)
	extCreateParams := component.ExtensionCreateParams{
		Logger:               zap.NewNop(),
		ApplicationStartInfo: component.DefaultApplicationStartInfo(),
	}

	var activeExt, builtExt component.Extension
	defer func() {
		// If the shutdown happens here there were already some errors on the test.
		// Ignore errors on this attempt to clean-up.
		if activeExt != nil {
			_ = activeExt.Shutdown(ctx)
		}
		if builtExt != nil {
			_ = builtExt.Shutdown(ctx)
		}
	}()

	if getConfigFn == nil {
		getConfigFn = factory.CreateDefaultConfig
	}

	for i := 0; i < 2; i++ {
		var err error
		builtExt, err = factory.CreateExtension(ctx, extCreateParams, getConfigFn())
		require.NoError(t, err, "Extension type: %s pass: %d", factory.Type(), i)

		if activeExt != nil {
			require.NoError(t, activeExt.Shutdown(ctx), "Extension type: %s pass: %d", factory.Type(), i)
			activeExt = nil
		}

		require.NoError(t, builtExt.Start(ctx, host), "Extension type: %s pass: %d", factory.Type(), i)
		activeExt = builtExt
		builtExt = nil
	}

	// Success path for shutdown.
	if activeExt != nil {
		assert.NoError(t, activeExt.Shutdown(ctx), "Extension type: %s", factory.Type())
		activeExt = nil
	}
}
