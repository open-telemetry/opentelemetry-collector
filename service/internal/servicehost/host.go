// Copyright  The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package servicehost

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
	ReportComponentStatus(source component.StatusSource, event *component.StatusEvent)

	// See component.Host for the documentation of the rest of the functions.

	// Deprecated: [0.65.0] Replaced by ReportComponentStatus.
	ReportFatalError(err error)

	GetFactory(kind component.Kind, componentType component.Type) component.Factory
	GetExtensions() map[component.ID]component.Component
	GetExporters() map[component.DataType]map[component.ID]component.Component
}
