// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensioncapabilities // import "go.opentelemetry.io/collector/extension/extensioncapabilities"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/pipeline"
)

// Watcher is an extra interface for Extension hosted by the OpenTelemetry
// Collector that is to be implemented by extensions interested in changes to component
// status.
type Watcher interface {
	// ComponentStatusChanged notifies about a change in the source component status.
	// Extensions that implement this interface must be ready that the ComponentStatusChanged
	// may be called before, after or concurrently with calls to Component.Start() and Component.Shutdown().
	// The function may be called concurrently with itself.
	ComponentStatusChanged(componentID component.ID, kind component.Kind, pipelineIDs []pipeline.ID, event *componentstatus.Event)
}
