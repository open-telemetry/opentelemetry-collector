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

package component

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/config"
)

type nopExtension struct {
	StartFunc
	ShutdownFunc
}

func TestNewExtensionFactory(t *testing.T) {
	const typeStr = "test"
	defaultCfg := config.NewExtensionSettings(config.NewComponentID(typeStr))
	nopExtensionInstance := new(nopExtension)

	factory := NewExtensionFactory(
		typeStr,
		func() config.Extension { return &defaultCfg },
		func(ctx context.Context, settings ExtensionCreateSettings, extension config.Extension) (Extension, error) {
			return nopExtensionInstance, nil
		})
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())
	ext, err := factory.CreateExtension(context.Background(), ExtensionCreateSettings{}, &defaultCfg)
	assert.NoError(t, err)
	assert.Same(t, nopExtensionInstance, ext)
}
