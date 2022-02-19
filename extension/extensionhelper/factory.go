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

package extensionhelper // import "go.opentelemetry.io/collector/extension/extensionhelper"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
)

// Deprecated: [v0.45.0] not needed, will be removed soon.
type FactoryOption struct{}

// Deprecated: [v0.45.0] use component.ExtensionDefaultConfigFunc.
type CreateDefaultConfig = component.ExtensionDefaultConfigFunc

// Deprecated: [v0.45.0] use component.CreateExtensionFunc.
type CreateServiceExtension = component.CreateExtensionFunc

// Deprecated: [v0.45.0] use component.NewExtensionFactory.
func NewFactory(
	cfgType config.Type,
	createDefaultConfig component.ExtensionDefaultConfigFunc,
	createServiceExtension component.CreateExtensionFunc,
	options ...FactoryOption) component.ExtensionFactory {
	return component.NewExtensionFactory(cfgType, createDefaultConfig, createServiceExtension)
}
