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

package goldendataset

// Start of PICT inputs for generating golden dataset ResourceSpans (pict_input_traces.txt)

// Input columns in pict_input_traces.txt
const (
	TracesColumnResource               = 0
	TracesColumnInstrumentationLibrary = 1
	TracesColumnSpans                  = 2
)

// PICTInputResource enum for the supported types of resource instances that can be generated.
type PICTInputResource string

const (
	ResourceEmpty     PICTInputResource = "Empty"
	ResourceVMOnPrem  PICTInputResource = "VMOnPrem"
	ResourceVMCloud   PICTInputResource = "VMCloud"
	ResourceK8sOnPrem PICTInputResource = "K8sOnPrem"
	ResourceK8sCloud  PICTInputResource = "K8sCloud"
	ResourceFaas      PICTInputResource = "Faas"
	ResourceExec      PICTInputResource = "Exec"
)

// PICTInputInstrumentationLibrary enum for the number and kind of instrumentation library instances that can be generated.
type PICTInputInstrumentationLibrary string

const (
	LibraryNone PICTInputInstrumentationLibrary = "None"
	LibraryOne  PICTInputInstrumentationLibrary = "One"
	LibraryTwo  PICTInputInstrumentationLibrary = "Two"
)

// PICTInputSpans enum for the relative sizes of tracing spans that can be attached to an instrumentation library span instance.
type PICTInputSpans string

const (
	LibrarySpansNone    PICTInputSpans = "None"
	LibrarySpansOne     PICTInputSpans = "One"
	LibrarySpansSeveral PICTInputSpans = "Several"
	LibrarySpansAll     PICTInputSpans = "All"
)

// PICTTracingInputs defines one pairwise combination of ResourceSpans variations
type PICTTracingInputs struct {
	// Specifies the category of attributes to populate the Resource field with
	Resource PICTInputResource
	// Specifies the number and library categories to populte the InstrumentationLibrarySpans field with
	InstrumentationLibrary PICTInputInstrumentationLibrary
	// Specifies the relative number of spans to populate the InstrumentationLibrarySpans' Spans field with
	Spans PICTInputSpans
}

// Start of PICT inputs for generating golden dataset Spans (pict_input_spans.txt)

// Input columns in pict_input_spans.txt
const (
	SpansColumnParent     = 0
	SpansColumnTracestate = 1
	SpansColumnKind       = 2
	SpansColumnAttributes = 3
	SpansColumnEvents     = 4
	SpansColumnLinks      = 5
	SpansColumnStatus     = 6
)

// PICTInputParent enum for the parent/child types of spans that can be generated.
type PICTInputParent string

const (
	SpanParentRoot  PICTInputParent = "Root"
	SpanParentChild PICTInputParent = "Child"
)

// PICTInputTracestate enum for the categories of tracestate values that can be generated for a span.
type PICTInputTracestate string

const (
	TraceStateEmpty PICTInputTracestate = "Empty"
	TraceStateOne   PICTInputTracestate = "One"
	TraceStateFour  PICTInputTracestate = "Four"
)

// PICTInputKind enum for the span kind values that can be set for a span.
type PICTInputKind string

const (
	SpanKindUnspecified PICTInputKind = "Unspecified"
	SpanKindInternal    PICTInputKind = "Internal"
	SpanKindServer      PICTInputKind = "Server"
	SpanKindClient      PICTInputKind = "Client"
	SpanKindProducer    PICTInputKind = "Producer"
	SpanKindConsumer    PICTInputKind = "Consumer"
)

// PICTInputAttributes enum for the categories of representative attributes a generated span can be populated with.
type PICTInputAttributes string

const (
	SpanAttrEmpty             PICTInputAttributes = "Empty"
	SpanAttrDatabaseSQL       PICTInputAttributes = "DatabaseSQL"
	SpanAttrDatabaseNoSQL     PICTInputAttributes = "DatabaseNoSQL"
	SpanAttrFaaSDatasource    PICTInputAttributes = "FaaSDatasource"
	SpanAttrFaaSHTTP          PICTInputAttributes = "FaaSHTTP"
	SpanAttrFaaSPubSub        PICTInputAttributes = "FaaSPubSub"
	SpanAttrFaaSTimer         PICTInputAttributes = "FaaSTimer"
	SpanAttrFaaSOther         PICTInputAttributes = "FaaSOther"
	SpanAttrHTTPClient        PICTInputAttributes = "HTTPClient"
	SpanAttrHTTPServer        PICTInputAttributes = "HTTPServer"
	SpanAttrMessagingProducer PICTInputAttributes = "MessagingProducer"
	SpanAttrMessagingConsumer PICTInputAttributes = "MessagingConsumer"
	SpanAttrGRPCClient        PICTInputAttributes = "gRPCClient"
	SpanAttrGRPCServer        PICTInputAttributes = "gRPCServer"
	SpanAttrInternal          PICTInputAttributes = "Internal"
	SpanAttrMaxCount          PICTInputAttributes = "MaxCount"
)

// PICTInputSpanChild enum for the categories of events and/or links a generated span can be populated with.
type PICTInputSpanChild string

const (
	SpanChildCountEmpty PICTInputSpanChild = "Empty"
	SpanChildCountOne   PICTInputSpanChild = "One"
	SpanChildCountTwo   PICTInputSpanChild = "Two"
	SpanChildCountEight PICTInputSpanChild = "Eight"
)

// PICTInputStatus enum for the status values a generated span can be populated with.
type PICTInputStatus string

const (
	SpanStatusUnset PICTInputStatus = "Unset"
	SpanStatusOk    PICTInputStatus = "Ok"
	SpanStatusError PICTInputStatus = "Error"
)

// PICTSpanInputs defines one pairwise combination of Span variations
type PICTSpanInputs struct {
	// Specifies whether the ParentSpanId field should be populated or not
	Parent PICTInputParent
	// Specifies the category of contents to populate the TraceState field with
	Tracestate PICTInputTracestate
	// Specifies the value to populate the Kind field with
	Kind PICTInputKind
	// Specifies the category of values to populate the Attributes field with
	Attributes PICTInputAttributes
	// Specifies the category of contents to populate the Events field with
	Events PICTInputSpanChild
	// Specifies the category of contents to populate the Links field with
	Links PICTInputSpanChild
	// Specifies the value to populate the Status field with
	Status PICTInputStatus
}
