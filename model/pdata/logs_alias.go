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

package pdata // import "go.opentelemetry.io/collector/model/pdata"

// This file contains aliases for log data structures.

import (
	"go.opentelemetry.io/collector/model/internal/pdata"
)

// LogsMarshaler is an alias for pdata.LogsMarshaler interface.
// Deprecated: [v0.49.0] Use logs.Marshaler instead.
type LogsMarshaler = pdata.LogsMarshaler

// LogsUnmarshaler is an alias for pdata.LogsUnmarshaler interface.
// Deprecated: [v0.49.0] Use logs.Unmarshaler instead.
type LogsUnmarshaler = pdata.LogsUnmarshaler

// LogsSizer is an alias for pdata.LogsSizer interface.
// Deprecated: [v0.49.0] Use logs.Sizer instead.
type LogsSizer = pdata.LogsSizer

// Logs is an alias for pdata.Logs struct.
// Deprecated: [v0.49.0] Use logs.Logs instead.
type Logs = pdata.Logs

// NewLogs is an alias for a function to create new Logs.
// Deprecated: [v0.49.0] Use logs.New instead.
var NewLogs = pdata.NewLogs

// SeverityNumber is an alias for pdata.SeverityNumber type.
// Deprecated: [v0.49.0] Use logs.SeverityNumber instead.
type SeverityNumber = pdata.SeverityNumber

const (

	// Deprecated: [v0.49.0] Use logs.SeverityNumberUNDEFINED instead.
	SeverityNumberUNDEFINED = pdata.SeverityNumberUNDEFINED

	// Deprecated: [v0.49.0] Use logs.SeverityNumberTRACE instead.
	SeverityNumberTRACE = pdata.SeverityNumberTRACE

	// Deprecated: [v0.49.0] Use logs.SeverityNumberTRACE2 instead.
	SeverityNumberTRACE2 = pdata.SeverityNumberTRACE2

	// Deprecated: [v0.49.0] Use logs.SeverityNumberTRACE3 instead.
	SeverityNumberTRACE3 = pdata.SeverityNumberTRACE3

	// Deprecated: [v0.49.0] Use logs.SeverityNumberTRACE4 instead.
	SeverityNumberTRACE4 = pdata.SeverityNumberTRACE4

	// Deprecated: [v0.49.0] Use logs.SeverityNumberDEBUG instead.
	SeverityNumberDEBUG = pdata.SeverityNumberDEBUG

	// Deprecated: [v0.49.0] Use logs.SeverityNumberDEBUG2 instead.
	SeverityNumberDEBUG2 = pdata.SeverityNumberDEBUG2

	// Deprecated: [v0.49.0] Use logs.SeverityNumberDEBUG3 instead.
	SeverityNumberDEBUG3 = pdata.SeverityNumberDEBUG3

	// Deprecated: [v0.49.0] Use logs.SeverityNumberDEBUG4 instead.
	SeverityNumberDEBUG4 = pdata.SeverityNumberDEBUG4

	// Deprecated: [v0.49.0] Use logs.SeverityNumberINFO instead.
	SeverityNumberINFO = pdata.SeverityNumberINFO

	// Deprecated: [v0.49.0] Use logs.SeverityNumberINFO2 instead.
	SeverityNumberINFO2 = pdata.SeverityNumberINFO2

	// Deprecated: [v0.49.0] Use logs.SeverityNumberINFO3 instead.
	SeverityNumberINFO3 = pdata.SeverityNumberINFO3

	// Deprecated: [v0.49.0] Use logs.SeverityNumberINFO4 instead.
	SeverityNumberINFO4 = pdata.SeverityNumberINFO4

	// Deprecated: [v0.49.0] Use logs.SeverityNumberWARN instead.
	SeverityNumberWARN = pdata.SeverityNumberWARN

	// Deprecated: [v0.49.0] Use logs.SeverityNumberWARN2 instead.
	SeverityNumberWARN2 = pdata.SeverityNumberWARN2

	// Deprecated: [v0.49.0] Use logs.SeverityNumberWARN3 instead.
	SeverityNumberWARN3 = pdata.SeverityNumberWARN3

	// Deprecated: [v0.49.0] Use logs.SeverityNumberWARN4 instead.
	SeverityNumberWARN4 = pdata.SeverityNumberWARN4

	// Deprecated: [v0.49.0] Use logs.SeverityNumberERROR instead.
	SeverityNumberERROR = pdata.SeverityNumberERROR

	// Deprecated: [v0.49.0] Use logs.SeverityNumberERROR2 instead.
	SeverityNumberERROR2 = pdata.SeverityNumberERROR2

	// Deprecated: [v0.49.0] Use logs.SeverityNumberERROR3 instead.
	SeverityNumberERROR3 = pdata.SeverityNumberERROR3

	// Deprecated: [v0.49.0] Use logs.SeverityNumberERROR4 instead.
	SeverityNumberERROR4 = pdata.SeverityNumberERROR4

	// Deprecated: [v0.49.0] Use logs.SeverityNumberFATAL instead.
	SeverityNumberFATAL = pdata.SeverityNumberFATAL

	// Deprecated: [v0.49.0] Use logs.SeverityNumberFATAL2 instead.
	SeverityNumberFATAL2 = pdata.SeverityNumberFATAL2

	// Deprecated: [v0.49.0] Use logs.SeverityNumberFATAL3 instead.
	SeverityNumberFATAL3 = pdata.SeverityNumberFATAL3

	// Deprecated: [v0.49.0] Use logs.SeverityNumberFATAL4 instead.
	SeverityNumberFATAL4 = pdata.SeverityNumberFATAL4
)

// Deprecated: [v0.48.0] Use ScopeLogsSlice instead.
type InstrumentationLibraryLogsSlice = pdata.ScopeLogsSlice

// Deprecated: [v0.48.0] Use NewScopeLogsSlice instead.
var NewInstrumentationLibraryLogsSlice = pdata.NewScopeLogsSlice

// Deprecated: [v0.48.0] Use ScopeLogs instead.
type InstrumentationLibraryLogs = pdata.ScopeLogs

// Deprecated: [v0.48.0] Use NewScopeLogs instead.
var NewInstrumentationLibraryLogs = pdata.NewScopeLogs
