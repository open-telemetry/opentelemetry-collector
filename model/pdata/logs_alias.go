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

import "go.opentelemetry.io/collector/model/internal"

// LogsMarshaler is an alias for internal.LogsMarshaler interface.
type LogsMarshaler = internal.LogsMarshaler

// LogsUnmarshaler is an alias for internal.LogsUnmarshaler interface.
type LogsUnmarshaler = internal.LogsUnmarshaler

// LogsSizer is an alias for internal.LogsSizer interface.
type LogsSizer = internal.LogsSizer

// Logs is an alias for internal.Logs struct.
type Logs = internal.Logs

// NewLogs is an alias for a function to create new Logs.
var NewLogs = internal.NewLogs

// SeverityNumber is an alias for internal.SeverityNumber type.
type SeverityNumber = internal.SeverityNumber

const (
	SeverityNumberUNDEFINED = internal.SeverityNumberUNDEFINED
	SeverityNumberTRACE     = internal.SeverityNumberTRACE
	SeverityNumberTRACE2    = internal.SeverityNumberTRACE2
	SeverityNumberTRACE3    = internal.SeverityNumberTRACE3
	SeverityNumberTRACE4    = internal.SeverityNumberTRACE4
	SeverityNumberDEBUG     = internal.SeverityNumberDEBUG
	SeverityNumberDEBUG2    = internal.SeverityNumberDEBUG2
	SeverityNumberDEBUG3    = internal.SeverityNumberDEBUG3
	SeverityNumberDEBUG4    = internal.SeverityNumberDEBUG4
	SeverityNumberINFO      = internal.SeverityNumberINFO
	SeverityNumberINFO2     = internal.SeverityNumberINFO2
	SeverityNumberINFO3     = internal.SeverityNumberINFO3
	SeverityNumberINFO4     = internal.SeverityNumberINFO4
	SeverityNumberWARN      = internal.SeverityNumberWARN
	SeverityNumberWARN2     = internal.SeverityNumberWARN2
	SeverityNumberWARN3     = internal.SeverityNumberWARN3
	SeverityNumberWARN4     = internal.SeverityNumberWARN4
	SeverityNumberERROR     = internal.SeverityNumberERROR
	SeverityNumberERROR2    = internal.SeverityNumberERROR2
	SeverityNumberERROR3    = internal.SeverityNumberERROR3
	SeverityNumberERROR4    = internal.SeverityNumberERROR4
	SeverityNumberFATAL     = internal.SeverityNumberFATAL
	SeverityNumberFATAL2    = internal.SeverityNumberFATAL2
	SeverityNumberFATAL3    = internal.SeverityNumberFATAL3
	SeverityNumberFATAL4    = internal.SeverityNumberFATAL4
)

// Deprecated: [v0.48.0] Use ScopeLogsSlice instead.
type InstrumentationLibraryLogsSlice = internal.ScopeLogsSlice

// Deprecated: [v0.48.0] Use NewScopeLogsSlice instead.
var NewInstrumentationLibraryLogsSlice = internal.NewScopeLogsSlice

// Deprecated: [v0.48.0] Use ScopeLogs instead.
type InstrumentationLibraryLogs = internal.ScopeLogs

// Deprecated: [v0.48.0] Use NewScopeLogs instead.
var NewInstrumentationLibraryLogs = internal.NewScopeLogs
