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

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/internal/configsource/configmodels"
)

// ConfigSource is the interface to be implemented by objects used by the collector
// to retrieve external configuration information.
type ConfigSource interface {
	component.Component

	// BeginSession signals that the ConfigSource is about to be used to inject data into
	// the configuration. ConfigSource objects can assume that there won't be
	// concurrent sessions and can use this call according to their needs:
	// lock needed resources, suspend background tasks, etc.
	BeginSession(ctx context.Context) error

	// Apply retrieves generic values given the arguments specified on a specific
	// invocation of a configuration source. The returned object is injected on
	// configuration. When Apply is called params will be the value associated to
	// an invocation of the ConfigSource. For example, the following YAML:
	//
	//  $my_config_src:
	//    param0: true
	//    param1: "some string"
	//
	// will have a call to Apply in which params is:
	//
	//  map[string]interface{}{
	//    "param0": true,
	//    "param1": "some string",
	// }
	//
	// A ConfigSource implementation should unmarshall, or cast it, as appropriate
	// and report errors if it doesn't fit their usage expectation.
	Apply(ctx context.Context, params interface{}) (interface{}, error)

	// EndSession signals that the configuration was fully flattened and it
	// is ready to be loaded. Each ConfigSource can use this call according
	// to their needs: release resources, start background tasks, update
	// internal state, etc.
	EndSession(ctx context.Context)
}

// CreateConfigSourceParams is passed to ConfigSourceFactory.CreateConfigSource function.
type CreateConfigSourceParams struct {
	// Logger that the factory can use during creation and can pass to the created
	// component to be used later as well.
	Logger *zap.Logger

	// ApplicationStartInfo can be used to retrieve data according to version, etc.
	ApplicationStartInfo component.ApplicationStartInfo
}

// ConfigSourceFactory is a factory interface for configuration sources.
type ConfigSourceFactory interface {
	component.Factory

	// CreateDefaultConfig creates the default configuration for the ConfigSource.
	// This method can be called multiple times depending on the pipeline
	// configuration and should not cause side-effects that prevent the creation
	// of multiple instances of the ConfigSource.
	// The object returned by this method needs to pass the checks implemented by
	// 'configcheck.ValidateConfig'. It is recommended to have such check in the
	// tests of any implementation of the Factory interface.
	CreateDefaultConfig() configmodels.ConfigSource

	// CreateConfigSource creates a configuration source based on the given config.
	CreateConfigSource(ctx context.Context, params CreateConfigSourceParams, cfg configmodels.ConfigSource) (ConfigSource, error)
}
