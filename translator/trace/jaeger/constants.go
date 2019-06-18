// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package jaeger

import (
	"errors"
)

const (
	// Jaeger Tags
	ocTimeEventUnknownType           = "opentel.timeevent.unknown.type"
	ocTimeEventAnnotationDescription = "opentel.timeevent.annotation.description"
	ocTimeEventMessageEventType      = "opentel.timeevent.messageevent.type"
	ocTimeEventMessageEventID        = "opentel.timeevent.messageevent.id"
	ocTimeEventMessageEventUSize     = "opentel.timeevent.messageevent.usize"
	ocTimeEventMessageEventCSize     = "opentel.timeevent.messageevent.csize"
	ocSameProcessAsParentSpan        = "opentel.sameprocessasparentspan"
	ocSpanChildCount                 = "opentel.span.childcount"
	opentelemetryLanguage               = "opentelemtry.language"
	opentelemetryExporterVersion        = "opentelemtry.exporterversion"
	opentelemetryCoreLibVersion         = "opentelemtry.corelibversion"
)

var (
	errZeroTraceID     = errors.New("OC span has an all zeros trace ID")
	errNilTraceID      = errors.New("OC trace ID is nil")
	errWrongLenTraceID = errors.New("TraceID does not have 16 bytes")
	errZeroSpanID      = errors.New("OC span has an all zeros span ID")
	errNilID           = errors.New("OC ID is nil")
	errWrongLenID      = errors.New("ID does not have 8 bytes")
)
