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
	"go.opentelemetry.io/collector/model/plog"
)

// LogsMarshaler is an alias for plog.LogsMarshaler interface.
type LogsMarshaler = plog.LogsMarshaler

// LogsUnmarshaler is an alias for plog.LogsUnmarshaler interface.
type LogsUnmarshaler = plog.LogsUnmarshaler

// LogsSizer is an alias for plog.LogsSizer interface.
type LogsSizer = plog.LogsSizer

// Logs is an alias for plog.Logs struct.
type Logs = plog.Logs

// NewLogs is an alias for a function to create new Logs.
var NewLogs = plog.NewLogs

// LogsFromInternalRep is an alias for plog.LogsFromInternalRep function.
// TODO: Can be removed, internal plog.LogsFromInternalRep should be used instead.
var LogsFromInternalRep = plog.LogsFromInternalRep

// Deprecated: [v0.44.0] use LogRecordSlice
type LogSlice = LogRecordSlice

// Deprecated: [v0.44.0] use NewLogRecordSlice
var NewLogSlice = NewLogRecordSlice

// SeverityNumber is an alias for plog.SeverityNumber type.
type SeverityNumber = plog.SeverityNumber

const (
	SeverityNumberUNDEFINED = plog.SeverityNumberUNDEFINED
	SeverityNumberTRACE     = plog.SeverityNumberTRACE
	SeverityNumberTRACE2    = plog.SeverityNumberTRACE2
	SeverityNumberTRACE3    = plog.SeverityNumberTRACE3
	SeverityNumberTRACE4    = plog.SeverityNumberTRACE4
	SeverityNumberDEBUG     = plog.SeverityNumberDEBUG
	SeverityNumberDEBUG2    = plog.SeverityNumberDEBUG2
	SeverityNumberDEBUG3    = plog.SeverityNumberDEBUG3
	SeverityNumberDEBUG4    = plog.SeverityNumberDEBUG4
	SeverityNumberINFO      = plog.SeverityNumberINFO
	SeverityNumberINFO2     = plog.SeverityNumberINFO2
	SeverityNumberINFO3     = plog.SeverityNumberINFO3
	SeverityNumberINFO4     = plog.SeverityNumberINFO4
	SeverityNumberWARN      = plog.SeverityNumberWARN
	SeverityNumberWARN2     = plog.SeverityNumberWARN2
	SeverityNumberWARN3     = plog.SeverityNumberWARN3
	SeverityNumberWARN4     = plog.SeverityNumberWARN4
	SeverityNumberERROR     = plog.SeverityNumberERROR
	SeverityNumberERROR2    = plog.SeverityNumberERROR2
	SeverityNumberERROR3    = plog.SeverityNumberERROR3
	SeverityNumberERROR4    = plog.SeverityNumberERROR4
	SeverityNumberFATAL     = plog.SeverityNumberFATAL
	SeverityNumberFATAL2    = plog.SeverityNumberFATAL2
	SeverityNumberFATAL3    = plog.SeverityNumberFATAL3
	SeverityNumberFATAL4    = plog.SeverityNumberFATAL4
)
