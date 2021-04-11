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

package configsource

import (
	"context"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
)

// CreateParams is passed to Factory.Create* functions.
type CreateParams struct {
	// Logger that the factory can use during creation and can pass to the created
	// ConfigSource to be used later as well.
	Logger *zap.Logger

	// ApplicationStartInfo can be used to retrieve data according to version, etc.
	ApplicationStartInfo component.ApplicationStartInfo
}

// Factory is a factory interface for configuration sources.
type Factory interface {
	component.Factory

	// CreateDefaultConfig creates the default configuration settings for the ConfigSource.
	// This method can be called multiple times depending on the pipeline
	// configuration and should not cause side-effects that prevent the creation
	// of multiple instances of the ConfigSource.
	// The object returned by this method needs to pass the checks implemented by
	// 'configcheck.ValidateConfig'. It is recommended to have such check in the
	// tests of any implementation of the Factory interface.
	CreateDefaultConfig() ConfigSettings

	// CreateConfigSource creates a configuration source based on the given config.
	CreateConfigSource(ctx context.Context, params CreateParams, cfg ConfigSettings) (ConfigSource, error)
}

// Factories maps the type of a ConfigSource to the respective factory object.
type Factories map[config.Type]Factory
