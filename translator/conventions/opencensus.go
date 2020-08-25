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

package conventions

// OTLP attributes to map certain OpenCensus proto fields. These fields don't have
// corresponding fields in OTLP, nor are defined in OTLP semantic conventions.
// TODO: decide if any of these must be in OTLP semantic conventions.
const (
	OCAttributeProcessStartTime        = "opencensus.starttime"
	OCAttributeProcessID               = "opencensus.pid"
	OCAttributeExporterVersion         = "opencensus.exporterversion"
	OCAttributeResourceType            = "opencensus.resourcetype"
	OCAttributeSameProcessAsParentSpan = "opencensus.same_process_as_parent_span"
	OCTimeEventMessageEventType        = "opencensus.timeevent.messageevent.type"
	OCTimeEventMessageEventID          = "opencensus.timeevent.messageevent.id"
	OCTimeEventMessageEventUSize       = "opencensus.timeevent.messageevent.usize"
	OCTimeEventMessageEventCSize       = "opencensus.timeevent.messageevent.csize"
)
