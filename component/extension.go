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

package component // import "go.opentelemetry.io/collector/component"

import (
	"context"
)

// Deprecated: [v0.67.0] use extension.Extension.
type Extension = Component

// Deprecated: [v0.67.0] use extension.CreateSettings.
type ExtensionCreateSettings struct {
	// ID returns the ID of the component that will be created.
	ID ID

	TelemetrySettings

	// BuildInfo can be used by components for informational purposes
	BuildInfo BuildInfo
}

// Deprecated: [v0.67.0] use extension.Factory.
type ExtensionFactory interface {
	Factory

	// CreateExtension creates an extension based on the given config.
	CreateExtension(ctx context.Context, set ExtensionCreateSettings, cfg Config) (Extension, error)

	// ExtensionStability gets the stability level of the Extension.
	ExtensionStability() StabilityLevel
}
