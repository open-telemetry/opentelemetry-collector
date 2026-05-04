// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph // import "go.opentelemetry.io/collector/service/internal/graph"

import (
	"go.opentelemetry.io/collector/component"
)

// componentNode is a utility interface to get the component ID of receivers, exporters, processors and connectors
type componentNode interface {
	ComponentID() component.ID
}
