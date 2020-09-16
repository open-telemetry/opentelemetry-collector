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
	"testing"

	gogoproto "github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/internal"
	otlpcollectorlogs "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/collector/logs/v1"
	otlplogs "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/logs/v1"
)

func TestLogRecordCount(t *testing.T) {
	md := NewLogs()
	assert.EqualValues(t, 0, md.LogRecordCount())

	md.ResourceLogs().Resize(1)
	assert.EqualValues(t, 0, md.LogRecordCount())

	md.ResourceLogs().At(0).InstrumentationLibraryLogs().Resize(1)
	assert.EqualValues(t, 0, md.LogRecordCount())

	md.ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).Logs().Resize(1)
	assert.EqualValues(t, 1, md.LogRecordCount())

	rms := md.ResourceLogs()
	rms.Resize(3)
	rms.At(0).InstrumentationLibraryLogs().Resize(1)
	rms.At(0).InstrumentationLibraryLogs().At(0).Logs().Resize(1)
	rms.At(1).InstrumentationLibraryLogs().Resize(1)
	rms.At(2).InstrumentationLibraryLogs().Resize(1)
	rms.At(2).InstrumentationLibraryLogs().At(0).Logs().Resize(5)
	assert.EqualValues(t, 6, md.LogRecordCount())
}

func TestLogRecordCountWithNils(t *testing.T) {
	assert.EqualValues(t, 0, LogsFromInternalRep(internal.LogsFromOtlp([]*otlplogs.ResourceLogs{nil, {}})).LogRecordCount())
	assert.EqualValues(t, 0, LogsFromInternalRep(internal.LogsFromOtlp([]*otlplogs.ResourceLogs{
		{
			InstrumentationLibraryLogs: []*otlplogs.InstrumentationLibraryLogs{nil, {}},
		},
	})).LogRecordCount())
	assert.EqualValues(t, 2, LogsFromInternalRep(internal.LogsFromOtlp([]*otlplogs.ResourceLogs{
		{
			InstrumentationLibraryLogs: []*otlplogs.InstrumentationLibraryLogs{
				{
					Logs: []*otlplogs.LogRecord{nil, {}},
				},
			},
		},
	})).LogRecordCount())
}

func TestToFromLogProto(t *testing.T) {
	otlp := []*otlplogs.ResourceLogs(nil)
	td := LogsFromInternalRep(internal.LogsFromOtlp(otlp))
	assert.EqualValues(t, NewLogs(), td)
	assert.EqualValues(t, otlp, *td.orig)
}

func TestLogs_ToOtlpProtoBytes(t *testing.T) {
	otlp := []*otlplogs.ResourceLogs(nil)
	ld := LogsFromInternalRep(internal.LogsFromOtlp(otlp))
	bytes, err := ld.ToOtlpProtoBytes()
	assert.Nil(t, err)

	elsr := otlpcollectorlogs.ExportLogsServiceRequest{}
	err = gogoproto.Unmarshal(bytes, &elsr)
	assert.Nil(t, err)
	assert.EqualValues(t, elsr.ResourceLogs, *ld.orig)
}
