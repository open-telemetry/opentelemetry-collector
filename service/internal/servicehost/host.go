// Copyright  The OpenTelemetry Authors
//
// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package servicehost // import "go.opentelemetry.io/collector/service/internal/servicehost"

import (
	"go.opentelemetry.io/collector/component"
)

// Host mirrors component.Host interface, with one important difference: servicehost.Host
// is not associated with a component and thus ReportComponentStatus() requires the source
// component to be explicitly specified.
type Host interface {
	// ReportComponentStatus is used to communicate the status of a source component to the Host.
	// The Host implementations will broadcast this information to interested parties via
	// StatusWatcher interface.
	ReportComponentStatus(source *component.InstanceID, event *component.StatusEvent)

	// See component.Host for the documentation of the rest of the functions.

	// Deprecated: [0.65.0] Replaced by ReportComponentStatus.
	ReportFatalError(err error)

	GetFactory(kind component.Kind, componentType component.Type) component.Factory
	GetExtensions() map[component.ID]component.Component
	GetExporters() map[component.DataType]map[component.ID]component.Component
}
