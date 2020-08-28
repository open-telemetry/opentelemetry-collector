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

package pdata

import (
	otlpcommon "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/common/v1"
	otlpresource "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/resource/v1"
)

// Metrics is an opaque interface that allows transition to the new internal Metrics data, but also facilitate the
// transition to the new components especially for traces.
//
// Outside of the core repository the metrics pipeline cannot be converted to the new model since data.MetricData is
// part of the internal package.
//
// IMPORTANT: Do not try to convert to/from this manually, use the helper functions in the pdatautil instead.
type Metrics struct {
	InternalOpaque interface{}
}

// DeprecatedNewResource temporary public function.
func DeprecatedNewResource(orig **otlpresource.Resource) Resource {
	return newResource(orig)
}

// DeprecatedNewInstrumentationLibrary temporary public function.
func DeprecatedNewInstrumentationLibrary(orig **otlpcommon.InstrumentationLibrary) InstrumentationLibrary {
	return newInstrumentationLibrary(orig)
}

// DeprecatedNewStringMap temporary public function.
func DeprecatedNewStringMap(orig *[]*otlpcommon.StringKeyValue) StringMap {
	return newStringMap(orig)
}
