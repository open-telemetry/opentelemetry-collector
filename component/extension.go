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

	"go.opentelemetry.io/collector/config/configmodels"
)

// ServiceExtension is the interface for objects hosted by the OpenTelemetry Collector that
// don't participate directly on data pipelines but provide some functionality
// to the service, examples: health check endpoint, z-pages, etc.
type ServiceExtension interface {
	Component
}

// PipelineWatcher is an extra interface for ServiceExtension hosted by the OpenTelemetry
// Collector that is to be implemented by extensions interested in changes to pipeline
// states. Typically this will be used by extensions that change their behavior if data is
// being ingested or not, e.g.: a k8s readiness probe.
type PipelineWatcher interface {
	// Ready notifies the ServiceExtension that all pipelines were built and the
	// receivers were started, i.e.: the service is ready to receive data
	// (notice that it may already have received data when this method is called).
	Ready() error

	// NotReady notifies the ServiceExtension that all receivers are about to be stopped,
	// i.e.: pipeline receivers will not accept new data.
	// This is sent before receivers are stopped, so the ServiceExtension can take any
	// appropriate action before that happens.
	NotReady() error
}

// ExtensionCreateParams is passed to ExtensionFactory.Create* functions.
type ExtensionCreateParams struct {
	// Logger that the factory can use during creation and can pass to the created
	// component to be used later as well.
	Logger *zap.Logger

	// ApplicationStartInfo can be used by components for informational purposes
	ApplicationStartInfo ApplicationStartInfo
}

// ExtensionFactory is a factory interface for extensions to the service.
type ExtensionFactory interface {
	Factory

	// CreateDefaultConfig creates the default configuration for the Extension.
	// This method can be called multiple times depending on the pipeline
	// configuration and should not cause side-effects that prevent the creation
	// of multiple instances of the Extension.
	// The object returned by this method needs to pass the checks implemented by
	// 'configcheck.ValidateConfig'. It is recommended to have such check in the
	// tests of any implementation of the Factory interface.
	CreateDefaultConfig() configmodels.Extension

	// CreateExtension creates a service extension based on the given config.
	CreateExtension(ctx context.Context, params ExtensionCreateParams, cfg configmodels.Extension) (ServiceExtension, error)
}
