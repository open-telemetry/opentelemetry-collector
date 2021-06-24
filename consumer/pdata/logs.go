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
	"fmt"

	"go.opentelemetry.io/collector/internal"
	otlpcollectorlog "go.opentelemetry.io/collector/internal/data/protogen/collector/logs/v1"
	otlplogs "go.opentelemetry.io/collector/internal/data/protogen/logs/v1"
)

// LogsDecoder is an interface to decode bytes into protocol-specific data model.
type LogsDecoder interface {
	// DecodeLogs decodes bytes into protocol-specific data model.
	// If the error is not nil, the returned interface cannot be used.
	DecodeLogs(buf []byte) (interface{}, error)
}

// LogsEncoder is an interface to encode protocol-specific data model into bytes.
type LogsEncoder interface {
	// EncodeLogs encodes protocol-specific data model into bytes.
	// If the error is not nil, the returned bytes slice cannot be used.
	EncodeLogs(model interface{}) ([]byte, error)
}

// FromLogsTranslator is an interface to translate pdata.Logs into protocol-specific data model.
type FromLogsTranslator interface {
	// FromLogs translates pdata.Logs into protocol-specific data model.
	// If the error is not nil, the returned pdata.Logs cannot be used.
	FromLogs(ld Logs) (interface{}, error)
}

// ToLogsTranslator is an interface to translate a protocol-specific data model into pdata.Traces.
type ToLogsTranslator interface {
	// ToLogs translates a protocol-specific data model into pdata.Logs.
	// If the error is not nil, the returned pdata.Logs cannot be used.
	ToLogs(src interface{}) (Logs, error)
}

// LogsMarshaler marshals pdata.Logs into bytes.
type LogsMarshaler interface {
	// Marshal the given pdata.Logs into bytes.
	// If the error is not nil, the returned bytes slice cannot be used.
	Marshal(td Logs) ([]byte, error)
}

type logsMarshaler struct {
	encoder    LogsEncoder
	translator FromLogsTranslator
}

// NewLogsMarshaler returns a new LogsMarshaler.
func NewLogsMarshaler(encoder LogsEncoder, translator FromLogsTranslator) LogsMarshaler {
	return &logsMarshaler{
		encoder:    encoder,
		translator: translator,
	}
}

// Marshal pdata.Logs into bytes.
func (t *logsMarshaler) Marshal(td Logs) ([]byte, error) {
	model, err := t.translator.FromLogs(td)
	if err != nil {
		return nil, fmt.Errorf("converting pdata to model failed: %w", err)
	}
	buf, err := t.encoder.EncodeLogs(model)
	if err != nil {
		return nil, fmt.Errorf("marshal failed: %w", err)
	}
	return buf, nil
}

// LogsUnmarshaler unmarshalls bytes into pdata.Logs.
type LogsUnmarshaler interface {
	// Unmarshal the given bytes into pdata.Logs.
	// If the error is not nil, the returned pdata.Logs cannot be used.
	Unmarshal(buf []byte) (Logs, error)
}

type logsUnmarshaler struct {
	decoder    LogsDecoder
	translator ToLogsTranslator
}

// NewLogsUnmarshaler returns a new LogsUnmarshaler.
func NewLogsUnmarshaler(decoder LogsDecoder, translator ToLogsTranslator) LogsUnmarshaler {
	return &logsUnmarshaler{
		decoder:    decoder,
		translator: translator,
	}
}

// Unmarshal bytes into pdata.Logs. On error pdata.Logs is invalid.
func (t *logsUnmarshaler) Unmarshal(buf []byte) (Logs, error) {
	model, err := t.decoder.DecodeLogs(buf)
	if err != nil {
		return Logs{}, fmt.Errorf("unmarshal failed: %w", err)
	}
	td, err := t.translator.ToLogs(model)
	if err != nil {
		return Logs{}, fmt.Errorf("converting model to pdata failed: %w", err)
	}
	return td, nil
}

// Logs is the top-level struct that is propagated through the logs pipeline.
//
// This is a reference type (like builtin map).
//
// Must use NewLogs functions to create new instances.
// Important: zero-initialized instance is not valid for use.
type Logs struct {
	orig *otlpcollectorlog.ExportLogsServiceRequest
}

// NewLogs creates a new Logs.
func NewLogs() Logs {
	return Logs{orig: &otlpcollectorlog.ExportLogsServiceRequest{}}
}

// LogsFromInternalRep creates the internal Logs representation from the ProtoBuf. Should
// not be used outside this module. This is intended to be used only by OTLP exporter and
// File exporter, which legitimately need to work with OTLP Protobuf structs.
func LogsFromInternalRep(logs internal.LogsWrapper) Logs {
	return Logs{orig: internal.LogsToOtlp(logs)}
}

// LogsFromOtlpProtoBytes converts OTLP Collector ExportLogsServiceRequest
// ProtoBuf bytes to the internal Logs.
//
// Returns an invalid Logs instance if error is not nil.
func LogsFromOtlpProtoBytes(data []byte) (Logs, error) {
	req := otlpcollectorlog.ExportLogsServiceRequest{}
	if err := req.Unmarshal(data); err != nil {
		return Logs{}, err
	}
	return Logs{orig: &req}, nil
}

// InternalRep returns internal representation of the logs. Should not be used outside
// this module. This is intended to be used only by OTLP exporter and File exporter,
// which legitimately need to work with OTLP Protobuf structs.
func (ld Logs) InternalRep() internal.LogsWrapper {
	return internal.LogsFromOtlp(ld.orig)
}

// ToOtlpProtoBytes converts this Logs to the OTLP Collector ExportLogsServiceRequest
// ProtoBuf bytes.
//
// Returns an nil byte-array if error is not nil.
func (ld Logs) ToOtlpProtoBytes() ([]byte, error) {
	return ld.orig.Marshal()
}

// Clone returns a copy of Logs.
func (ld Logs) Clone() Logs {
	cloneLd := NewLogs()
	ld.ResourceLogs().CopyTo(cloneLd.ResourceLogs())
	return cloneLd
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

// OtlpProtoSize returns the size in bytes of this Logs encoded as OTLP Collector
// ExportLogsServiceRequest ProtoBuf bytes.
func (ld Logs) OtlpProtoSize() int {
	return ld.orig.Size()
}

// ResourceLogs returns the ResourceLogsSlice associated with this Logs.
func (ld Logs) ResourceLogs() ResourceLogsSlice {
	return newResourceLogsSlice(&ld.orig.ResourceLogs)
}

// SeverityNumber is the public alias of otlplogs.SeverityNumber from internal package.
type SeverityNumber int32

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

// String returns the string representation of the SeverityNumber.
func (sn SeverityNumber) String() string { return otlplogs.SeverityNumber(sn).String() }
