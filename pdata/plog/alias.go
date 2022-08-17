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

package plog // import "go.opentelemetry.io/collector/pdata/plog"

// This file contains aliases for logs data structures.

import "go.opentelemetry.io/collector/pdata/internal"

// Logs is the top-level struct that is propagated through the logs pipeline.
// Use NewLogs to create new instance, zero-initialized instance is not valid for use.
type Logs = internal.Logs

// NewLogs creates a new Logs struct.
var NewLogs = internal.NewLogs

// SeverityNumber represents severity number of a log record.
type SeverityNumber = internal.SeverityNumber

const (
	SeverityNumberUndefined = internal.SeverityNumberUndefined
	SeverityNumberTrace     = internal.SeverityNumberTrace
	SeverityNumberTrace2    = internal.SeverityNumberTrace2
	SeverityNumberTrace3    = internal.SeverityNumberTrace3
	SeverityNumberTrace4    = internal.SeverityNumberTrace4
	SeverityNumberDebug     = internal.SeverityNumberDebug
	SeverityNumberDebug2    = internal.SeverityNumberDebug2
	SeverityNumberDebug3    = internal.SeverityNumberDebug3
	SeverityNumberDebug4    = internal.SeverityNumberDebug4
	SeverityNumberInfo      = internal.SeverityNumberInfo
	SeverityNumberInfo2     = internal.SeverityNumberInfo2
	SeverityNumberInfo3     = internal.SeverityNumberInfo3
	SeverityNumberInfo4     = internal.SeverityNumberInfo4
	SeverityNumberWarn      = internal.SeverityNumberWarn
	SeverityNumberWarn2     = internal.SeverityNumberWarn2
	SeverityNumberWarn3     = internal.SeverityNumberWarn3
	SeverityNumberWarn4     = internal.SeverityNumberWarn4
	SeverityNumberError     = internal.SeverityNumberError
	SeverityNumberError2    = internal.SeverityNumberError2
	SeverityNumberError3    = internal.SeverityNumberError3
	SeverityNumberError4    = internal.SeverityNumberError4
	SeverityNumberFatal     = internal.SeverityNumberFatal
	SeverityNumberFatal2    = internal.SeverityNumberFatal2
	SeverityNumberFatal3    = internal.SeverityNumberFatal3
	SeverityNumberFatal4    = internal.SeverityNumberFatal4
)

const (
	// Deprecated: [0.59.0] Use SeverityNumberUndefined instead
	SeverityNumberUNDEFINED = SeverityNumberUndefined

	// Deprecated: [0.59.0] Use SeverityNumberTrace instead
	SeverityNumberTRACE = SeverityNumberTrace

	// Deprecated: [0.59.0] Use SeverityNumberTrace2 instead
	SeverityNumberTRACE2 = SeverityNumberTrace2

	// Deprecated: [0.59.0] Use SeverityNumberTrace3 instead
	SeverityNumberTRACE3 = SeverityNumberTrace3

	// Deprecated: [0.59.0] Use SeverityNumberTrace4 instead
	SeverityNumberTRACE4 = SeverityNumberTrace4

	// Deprecated: [0.59.0] Use SeverityNumberDebug instead
	SeverityNumberDEBUG = SeverityNumberDebug

	// Deprecated: [0.59.0] Use SeverityNumberDebug2 instead
	SeverityNumberDEBUG2 = SeverityNumberDebug2

	// Deprecated: [0.59.0] Use SeverityNumberDebug3 instead
	SeverityNumberDEBUG3 = SeverityNumberDebug3

	// Deprecated: [0.59.0] Use SeverityNumberDebug4 instead
	SeverityNumberDEBUG4 = SeverityNumberDebug4

	// Deprecated: [0.59.0] Use SeverityNumberInfo instead
	SeverityNumberINFO = SeverityNumberInfo

	// Deprecated: [0.59.0] Use SeverityNumberInfo2 instead
	SeverityNumberINFO2 = SeverityNumberInfo2

	// Deprecated: [0.59.0] Use SeverityNumberInfo3 instead
	SeverityNumberINFO3 = SeverityNumberInfo3

	// Deprecated: [0.59.0] Use SeverityNumberInfo4 instead
	SeverityNumberINFO4 = SeverityNumberInfo4

	// Deprecated: [0.59.0] Use SeverityNumberWarn instead
	SeverityNumberWARN = SeverityNumberWarn

	// Deprecated: [0.59.0] Use SeverityNumberWarn2 instead
	SeverityNumberWARN2 = SeverityNumberWarn2

	// Deprecated: [0.59.0] Use SeverityNumberWarn3 instead
	SeverityNumberWARN3 = SeverityNumberWarn3

	// Deprecated: [0.59.0] Use SeverityNumberWarn4 instead
	SeverityNumberWARN4 = SeverityNumberWarn4

	// Deprecated: [0.59.0] Use SeverityNumberError instead
	SeverityNumberERROR = SeverityNumberError

	// Deprecated: [0.59.0] Use SeverityNumberError2 instead
	SeverityNumberERROR2 = SeverityNumberError2

	// Deprecated: [0.59.0] Use SeverityNumberError3 instead
	SeverityNumberERROR3 = SeverityNumberError3

	// Deprecated: [0.59.0] Use SeverityNumberError4 instead
	SeverityNumberERROR4 = SeverityNumberError4

	// Deprecated: [0.59.0] Use SeverityNumberFatal instead
	SeverityNumberFATAL = SeverityNumberFatal

	// Deprecated: [0.59.0] Use SeverityNumberFatal2 instead
	SeverityNumberFATAL2 = SeverityNumberFatal2

	// Deprecated: [0.59.0] Use SeverityNumberFatal3 instead
	SeverityNumberFATAL3 = SeverityNumberFatal3

	// Deprecated: [0.59.0] Use SeverityNumberFatal4 instead
	SeverityNumberFATAL4 = SeverityNumberFatal4
)
