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

// Package pdata (pipeline data) implements data structures that represent telemetry data in-memory.
// All data received is converted into this format and travels through the pipeline
// in this format and that is converted from this format by exporters when sending.
//
// Current implementation primarily uses OTLP ProtoBuf structs as the underlying data
// structures for many of of the declared structs. We keep a pointer to OTLP protobuf
// in the "orig" member field. This allows efficient translation to/from OTLP wire
// protocol. Note that the underlying data structure is kept private so that in the
// future we are free to make changes to it to make more optimal.
//
// Most of internal data structures must be created via New* functions. Zero-initialized
// structures in most cases are not valid (read comments for each struct to know if it
// is the case). This is a slight deviation from idiomatic Go to avoid unnecessary
// pointer checks in dozens of functions which assume the invariant that "orig" member
// is non-nil. Several structures also provide New*Slice functions that allows to create
// more than one instance of the struct more efficiently instead of calling New*
// repeatedly. Use it where appropriate.
package pdata
