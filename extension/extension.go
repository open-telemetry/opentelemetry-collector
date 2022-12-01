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

package extension // import "go.opentelemetry.io/collector/extension"

import (
	"go.opentelemetry.io/collector/component"
)

// Extension is the interface for objects hosted by the OpenTelemetry Collector that
// don't participate directly on data pipelines but provide some functionality
// to the service, examples: health check endpoint, z-pages, etc.
type Extension = component.Extension //nolint:staticcheck

// PipelineWatcher is an extra interface for Extension hosted by the OpenTelemetry
// Collector that is to be implemented by extensions interested in changes to pipeline
// states. Typically this will be used by extensions that change their behavior if data is
// being ingested or not, e.g.: a k8s readiness probe.
type PipelineWatcher = component.PipelineWatcher //nolint:staticcheck

// CreateSettings is passed to Factory.Create(...) function.
type CreateSettings = component.ExtensionCreateSettings //nolint:staticcheck

// CreateFunc is the equivalent of Factory.Create(...) function.
type CreateFunc = component.CreateExtensionFunc //nolint:staticcheck

// Factory is a factory for extensions to the service.
type Factory = component.ExtensionFactory //nolint:staticcheck

// NewFactory returns a new Factory  based on this configuration.
var NewFactory = component.NewExtensionFactory //nolint:staticcheck

// MakeFactoryMap takes a list of extension factories and returns a map
// with factory type as keys. It returns a non-nil error when more than one factories
// have the same type.
var MakeFactoryMap = component.MakeExtensionFactoryMap //nolint:staticcheck
