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

import (
	"go.opentelemetry.io/collector/pdata/internal"
	otlpcollectorlog "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/logs/v1"
	otlplogs "go.opentelemetry.io/collector/pdata/internal/data/protogen/logs/v1"
)

// Logs is the top-level struct that is propagated through the logs pipeline.
// Use NewLogs to create new instance, zero-initialized instance is not valid for use.
type Logs internal.Logs

func newLogs(orig *otlpcollectorlog.ExportLogsServiceRequest) Logs {
	return Logs(internal.NewLogs(orig))
}

func (ms Logs) getOrig() *otlpcollectorlog.ExportLogsServiceRequest {
	return internal.GetOrigLogs(internal.Logs(ms))
}

// NewLogs creates a new Logs struct.
func NewLogs() Logs {
	return newLogs(&otlpcollectorlog.ExportLogsServiceRequest{})
}

// MoveTo moves all properties from the current struct to dest
// resetting the current instance to its zero value.
func (ms Logs) MoveTo(dest Logs) {
	*dest.getOrig() = *ms.getOrig()
	*ms.getOrig() = otlpcollectorlog.ExportLogsServiceRequest{}
}

// Clone returns a copy of Logs.
func (ms Logs) Clone() Logs {
	cloneLd := NewLogs()
	ms.ResourceLogs().CopyTo(cloneLd.ResourceLogs())
	return cloneLd
}

// LogRecordCount calculates the total number of log records.
func (ms Logs) LogRecordCount() int {
	logCount := 0
	rss := ms.ResourceLogs()
	for i := 0; i < rss.Len(); i++ {
		rs := rss.At(i)
		ill := rs.ScopeLogs()
		for i := 0; i < ill.Len(); i++ {
			logs := ill.At(i)
			logCount += logs.LogRecords().Len()
		}
	}
	return logCount
}

// ResourceLogs returns the ResourceLogsSlice associated with this Logs.
func (ms Logs) ResourceLogs() ResourceLogsSlice {
	return newResourceLogsSlice(&ms.getOrig().ResourceLogs)
}

// SeverityNumber represents severity number of a log record.
type SeverityNumber int32

const (
	SeverityNumberUndefined = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_UNSPECIFIED)
	SeverityNumberTrace     = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_TRACE)
	SeverityNumberTrace2    = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_TRACE2)
	SeverityNumberTrace3    = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_TRACE3)
	SeverityNumberTrace4    = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_TRACE4)
	SeverityNumberDebug     = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_DEBUG)
	SeverityNumberDebug2    = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_DEBUG2)
	SeverityNumberDebug3    = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_DEBUG3)
	SeverityNumberDebug4    = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_DEBUG4)
	SeverityNumberInfo      = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_INFO)
	SeverityNumberInfo2     = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_INFO2)
	SeverityNumberInfo3     = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_INFO3)
	SeverityNumberInfo4     = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_INFO4)
	SeverityNumberWarn      = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_WARN)
	SeverityNumberWarn2     = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_WARN2)
	SeverityNumberWarn3     = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_WARN3)
	SeverityNumberWarn4     = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_WARN4)
	SeverityNumberError     = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_ERROR)
	SeverityNumberError2    = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_ERROR2)
	SeverityNumberError3    = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_ERROR3)
	SeverityNumberError4    = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_ERROR4)
	SeverityNumberFatal     = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_FATAL)
	SeverityNumberFatal2    = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_FATAL2)
	SeverityNumberFatal3    = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_FATAL3)
	SeverityNumberFatal4    = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_FATAL4)
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

// String returns the string representation of the SeverityNumber.
func (sn SeverityNumber) String() string { return otlplogs.SeverityNumber(sn).String() }

// Deprecated: [v0.59.0] use FlagsStruct().
func (ms LogRecord) Flags() uint32 {
	return ms.getOrig().Flags
}

// Deprecated: [v0.59.0] use FlagsStruct().
func (ms LogRecord) SetFlags(v uint32) {
	ms.getOrig().Flags = v
}
