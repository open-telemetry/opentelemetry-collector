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

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configmapprovider"
)

// ConfigSource is
type ConfigSource interface {
	configmapprovider.Provider
}

// ConfigSourceCreateSettings is passed to CreateConfigSource functions.
type ConfigSourceCreateSettings struct {
}

type ConfigSourceFactory interface {
	Factory
	CreateDefaultConfig() config.ConfigSource
	CreateConfigSource(ctx context.Context, set ConfigSourceCreateSettings, cfg config.ConfigSource) (configmapprovider.Provider, error)
}
