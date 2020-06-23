// Copyright The OpenTelemetry Authors
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

package tracetranslator

// Some of the keys used to represent OC proto constructs as tags or annotations in other formats.
const (
	AnnotationDescriptionKey = "description"

	MessageEventIDKey               = "message.id"
	MessageEventTypeKey             = "message.type"
	MessageEventCompressedSizeKey   = "message.compressed_size"
	MessageEventUncompressedSizeKey = "message.uncompressed_size"

	TagSpanKind = "span.kind"

	TagStatusCode          = "status.code"
	TagStatusMsg           = "status.message"
	TagError               = "error"
	TagHTTPStatusCode      = "http.status_code"
	TagHTTPStatusMsg       = "http.status_message"
	TagZipkinCensusCode    = "census.status_code"
	TagZipkinCensusMsg     = "census.status_description"
	TagZipkinOpenCensusMsg = "opencensus.status_description"
)

// OpenTracingSpanKind are possible values for TagSpanKind and match the OpenTracing
// conventions: https://github.com/opentracing/specification/blob/master/semantic_conventions.md
// These values are also used for representing internally span kinds that have no
// equivalents in OpenCensus format. They are stored as values of TagSpanKind
// Note: this internal usage needs to be eliminated when we move to OTLP for internal
// in-memory representation since OTLP has the equivalents.
type OpenTracingSpanKind string

const (
	OpenTracingSpanKindUnspecified OpenTracingSpanKind = ""
	OpenTracingSpanKindClient      OpenTracingSpanKind = "client"
	OpenTracingSpanKindServer      OpenTracingSpanKind = "server"
	OpenTracingSpanKindConsumer    OpenTracingSpanKind = "consumer"
	OpenTracingSpanKindProducer    OpenTracingSpanKind = "producer"
	OpenTracingSpanKindInternal    OpenTracingSpanKind = "internal"
)
