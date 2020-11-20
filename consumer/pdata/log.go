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
	"go.opentelemetry.io/collector/internal"
	otlpcollectorlog "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/collector/logs/v1"
	otlplogs "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/logs/v1"
)

// This file defines in-memory data structures to represent logs.

// Logs is the top-level struct that is propagated through the logs pipeline.
//
// This is a reference type (like builtin map).
//
// Must use NewLogs functions to create new instances.
// Important: zero-initialized instance is not valid for use.
type Logs struct {
	orig *[]*otlplogs.ResourceLogs
}

// NewLogs creates a new Logs.
func NewLogs() Logs {
	orig := []*otlplogs.ResourceLogs(nil)
	return Logs{&orig}
}

// LogsFromInternalRep creates the internal Logs representation from the ProtoBuf. Should
// not be used outside this module. This is intended to be used only by OTLP exporter and
// File exporter, which legitimately need to work with OTLP Protobuf structs.
func LogsFromInternalRep(logs internal.OtlpLogsWrapper) Logs {
	return Logs{logs.Orig}
}

// InternalRep returns internal representation of the logs. Should not be used outside
// this module. This is intended to be used only by OTLP exporter and File exporter,
// which legitimately need to work with OTLP Protobuf structs.
func (ld Logs) InternalRep() internal.OtlpLogsWrapper {
	return internal.OtlpLogsWrapper{Orig: ld.orig}
}

// ToOtlpProtoBytes returns the internal Logs to OTLP Collector ExportTraceServiceRequest
// ProtoBuf bytes. This is intended to export OTLP Protobuf bytes for OTLP/HTTP transports.
func (ld Logs) ToOtlpProtoBytes() ([]byte, error) {
	logs := otlpcollectorlog.ExportLogsServiceRequest{
		ResourceLogs: *ld.orig,
	}
	return logs.Marshal()
}

// FromOtlpProtoBytes converts OTLP Collector ExportLogsServiceRequest
// ProtoBuf bytes to the internal Logs. Overrides current data.
// Calling this function on zero-initialized structure causes panic.
// Use it with NewLogs or on existing initialized Logs.
func (ld Logs) FromOtlpProtoBytes(data []byte) error {
	logs := otlpcollectorlog.ExportLogsServiceRequest{}
	if err := logs.Unmarshal(data); err != nil {
		return err
	}
	*ld.orig = logs.ResourceLogs
	return nil
}

// Clone returns a copy of Logs.
func (ld Logs) Clone() Logs {
	rls := NewResourceLogsSlice()
	ld.ResourceLogs().CopyTo(rls)
	return Logs(rls)
}

// LogRecordCount calculates the total number of log records.
func (ld Logs) LogRecordCount() int {
	logCount := 0
	rss := ld.ResourceLogs()
	for i := 0; i < rss.Len(); i++ {
		rs := rss.At(i)
		ill := rs.InstrumentationLibraryLogs()
		for i := 0; i < ill.Len(); i++ {
			logs := ill.At(i)
			logCount += logs.Logs().Len()
		}
	}
	return logCount
}

// SizeBytes returns the number of bytes in the internal representation of the
// logs.
func (ld Logs) SizeBytes() int {
	size := 0
	for i := range *ld.orig {
		size += (*ld.orig)[i].Size()
	}
	return size
}

func (ld Logs) ResourceLogs() ResourceLogsSlice {
	return ResourceLogsSlice(ld)
}

// SeverityNumber is the public alias of otlplogs.SeverityNumber from internal package.
type SeverityNumber otlplogs.SeverityNumber

const (
	SeverityNumberUNDEFINED = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_UNSPECIFIED)
	SeverityNumberTRACE     = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_TRACE)
	SeverityNumberTRACE2    = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_TRACE2)
	SeverityNumberTRACE3    = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_TRACE3)
	SeverityNumberTRACE4    = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_TRACE4)
	SeverityNumberDEBUG     = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_DEBUG)
	SeverityNumberDEBUG2    = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_DEBUG2)
	SeverityNumberDEBUG3    = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_DEBUG3)
	SeverityNumberDEBUG4    = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_DEBUG4)
	SeverityNumberINFO      = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_INFO)
	SeverityNumberINFO2     = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_INFO2)
	SeverityNumberINFO3     = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_INFO3)
	SeverityNumberINFO4     = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_INFO4)
	SeverityNumberWARN      = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_WARN)
	SeverityNumberWARN2     = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_WARN2)
	SeverityNumberWARN3     = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_WARN3)
	SeverityNumberWARN4     = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_WARN4)
	SeverityNumberERROR     = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_ERROR)
	SeverityNumberERROR2    = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_ERROR2)
	SeverityNumberERROR3    = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_ERROR3)
	SeverityNumberERROR4    = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_ERROR4)
	SeverityNumberFATAL     = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_FATAL)
	SeverityNumberFATAL2    = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_FATAL2)
	SeverityNumberFATAL3    = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_FATAL3)
	SeverityNumberFATAL4    = SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_FATAL4)
)
