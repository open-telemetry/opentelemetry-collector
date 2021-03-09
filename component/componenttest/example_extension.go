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
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
)

// ExampleExtensionCfg is for testing purposes. We are defining an example config and factory
// for "exampleextension" extension type.
type ExampleExtensionCfg struct {
	configmodels.ExtensionSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
	ExtraSetting                   string                   `mapstructure:"extra"`
	ExtraMapSetting                map[string]string        `mapstructure:"extra_map"`
	ExtraListSetting               []string                 `mapstructure:"extra_list"`
}

type ExampleExtension struct {
}

func (e *ExampleExtension) Start(_ context.Context, _ component.Host) error { return nil }

func (e *ExampleExtension) Shutdown(_ context.Context) error { return nil }

// ExampleExtensionFactory is factory for ExampleExtensionCfg.
type ExampleExtensionFactory struct {
	FailCreation bool
}

// Type gets the type of the Extension config created by this factory.
func (f *ExampleExtensionFactory) Type() configmodels.Type {
	return "exampleextension"
}

// CreateDefaultConfig creates the default configuration for the Extension.
func (f *ExampleExtensionFactory) CreateDefaultConfig() configmodels.Extension {
	return &ExampleExtensionCfg{
		ExtensionSettings: configmodels.ExtensionSettings{
			TypeVal: f.Type(),
			NameVal: string(f.Type()),
		},
		ExtraSetting:     "extra string setting",
		ExtraMapSetting:  nil,
		ExtraListSetting: nil,
	}
}

// CreateExtension creates an Extension based on this config.
func (f *ExampleExtensionFactory) CreateExtension(_ context.Context, _ component.ExtensionCreateParams, _ configmodels.Extension) (component.Extension, error) {
	if f.FailCreation {
		return nil, fmt.Errorf("cannot create %q extension type", f.Type())
	}
	return &ExampleExtension{}, nil
}
