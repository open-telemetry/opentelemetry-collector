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
	// the configuration. The difference between BeginSession and the component.Start method
	// is that the latter is used by the host to manage the life-time of a ConfigSource and to
	// provide a way for a ConfigSource to have a reference to the host so it can notify the
	// host when configuration changes are detected.
	//
	// A ConfigSource should use the BeginSession call according to their needs:
	// lock resources, suspend background or watcher tasks, etc. An implementation, for
	// instance, can use the begin of a session to prevent torn configurations, by acquiring
	// a lock (or some other mechanism) that prevents concurrent changes to the configura during
	// the middle of a session.
	//
	// The code managing the session must guarantee that no ConfigSource instance participates
	// in concurrent sessions.
	BeginSession(ctx context.Context) error

	// Apply goes to the configuration source, sending the given parameters as a map
	// and returns the resulting configuration value or snippet. A configuration snippet
	// will become a map, as follows:
	//
	//  $my_config_src:
	//    param0: true
	//    param1: "some string"
	//
	// Becomes a call with the following payload as params:
	//
	//  map[string]interface{}{
	//    "param0": true,
	//    "param1": "some string",
	// }
	//
	// A ConfigSource should then unmarshal or cast the params. An error should be
	// returned when the params don't fit the expected usage.
	Apply(ctx context.Context, params interface{}) (interface{}, error)

	// EndSession signals that the configuration was fully flattened and it
	// is ready to be loaded. Each ConfigSource should use this call according
	// to their needs: release resources, start background or watcher tasks, update
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
