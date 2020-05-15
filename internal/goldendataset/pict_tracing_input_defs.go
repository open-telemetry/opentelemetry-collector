// Copyright 2020, OpenTelemetry Authors
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

package goldendataset

//// Start of PICT inputs for generating golden dataset ResourceSpans (pict_input_traces.txt) ////

// Input columns in pict_input_traces.txt
const (
	TracesColumnResource               = 0
	TracesColumnInstrumentationLibrary = 1
	TracesColumnSpans                  = 2
)

// Enumerates the supported types of resource instances that can be created
type PICTInputResource string

const (
	ResourceNil       PICTInputResource = "Nil"
	ResourceEmpty     PICTInputResource = "Empty"
	ResourceVMOnPrem  PICTInputResource = "VMOnPrem"
	ResourceVMCloud   PICTInputResource = "VMCloud"
	ResourceK8sOnPrem PICTInputResource = "K8sOnPrem"
	ResourceK8sCloud  PICTInputResource = "K8sCloud"
	ResourceFaas      PICTInputResource = "Faas"
)

// Enumerates the number and kind of instrumentation library instances that can be created
type PICTInputInstrumentationLibrary string

const (
	LibraryNone PICTInputInstrumentationLibrary = "None"
	LibraryOne  PICTInputInstrumentationLibrary = "One"
	LibraryTwo  PICTInputInstrumentationLibrary = "Two"
)

// Enumerates the relative sizes of tracing spans that can be attached to an instrumentation library span instance
type PICTInputSpans string

const (
	LibrarySpansNone    PICTInputSpans = "None"
	LibrarySpansOne     PICTInputSpans = "One"
	LibrarySpansSeveral PICTInputSpans = "Several"
	LibrarySpansAll     PICTInputSpans = "All"
)

//// Start of PICT inputs for generating golden dataset Spans (pict_input_spans.txt) ////

const (
	SpansColumnParent     = 0
	SpansColumnTracestate = 1
	SpansColumnKind       = 2
	SpansColumnAttributes = 3
	SpansColumnEvents     = 4
	SpansColumnLinks      = 5
	SpansColumnStatus     = 6
)

type PICTInputParent string

const (
	SpanParentRoot  PICTInputParent = "Root"
	SpanParentChild PICTInputParent = "Child"
)

type PICTInputTracestate string

const (
	TraceStateEmpty PICTInputTracestate = "Empty"
	TraceStateOne   PICTInputTracestate = "One"
	TraceStateFour  PICTInputTracestate = "Four"
)

type PICTInputKind string

const (
	SpanKindUnspecified PICTInputKind = "Unspecified"
	SpanKindInternal    PICTInputKind = "Internal"
	SpanKindServer      PICTInputKind = "Server"
	SpanKindClient      PICTInputKind = "Client"
	SpanKindProducer    PICTInputKind = "Producer"
	SpanKindConsumer    PICTInputKind = "Consumer"
)

type PICTInputAttributes string

const (
	SpanAttrNil               PICTInputAttributes = "Nil"
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
)

type PICTInputSpanChild string

const (
	SpanChildCountNil   PICTInputSpanChild = "Nil"
	SpanChildCountEmpty PICTInputSpanChild = "Empty"
	SpanChildCountOne   PICTInputSpanChild = "One"
	SpanChildCountTwo   PICTInputSpanChild = "Two"
	SpanChildCountEight PICTInputSpanChild = "Eight"
)

type PICTInputStatus string

const (
	SpanStatusNil                PICTInputStatus = "Nil"
	SpanStatusOk                 PICTInputStatus = "Ok"
	SpanStatusCancelled          PICTInputStatus = "Cancelled"
	SpanStatusUnknownError       PICTInputStatus = "UnknownError"
	SpanStatusInvalidArgument    PICTInputStatus = "InvalidArgument"
	SpanStatusDeadlineExceeded   PICTInputStatus = "DeadlineExceeded"
	SpanStatusNotFound           PICTInputStatus = "NotFound"
	SpanStatusAlreadyExists      PICTInputStatus = "AlreadyExists"
	SpanStatusPermissionDenied   PICTInputStatus = "PermissionDenied"
	SpanStatusResourceExhausted  PICTInputStatus = "ResourceExhausted"
	SpanStatusFailedPrecondition PICTInputStatus = "FailedPrecondition"
	SpanStatusAborted            PICTInputStatus = "Aborted"
	SpanStatusOutOfRange         PICTInputStatus = "OutOfRange"
	SpanStatusUnimplemented      PICTInputStatus = "Unimplemented"
	SpanStatusInternalError      PICTInputStatus = "InternalError"
	SpanStatusUnavailable        PICTInputStatus = "Unavailable"
	SpanStatusDataLoss           PICTInputStatus = "DataLoss"
	SpanStatusUnauthenticated    PICTInputStatus = "Unauthenticated"
)
