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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/internal"
	otlpcollectorlog "go.opentelemetry.io/collector/internal/data/protogen/collector/logs/v1"
	otlplogs "go.opentelemetry.io/collector/internal/data/protogen/logs/v1"
)

func TestLogsMarshal_TranslationError(t *testing.T) {
	translator := &mockTranslator{}
	encoder := &mockEncoder{}

	lm := NewLogsMarshaler(encoder, translator)
	ld := NewLogs()

	translator.On("FromLogs", ld).Return(nil, errors.New("translation failed"))

	_, err := lm.Marshal(ld)
	assert.Error(t, err)
	assert.EqualError(t, err, "converting pdata to model failed: translation failed")
}

func TestLogsMarshal_SerializeError(t *testing.T) {
	translator := &mockTranslator{}
	encoder := &mockEncoder{}

	lm := NewLogsMarshaler(encoder, translator)
	ld := NewLogs()
	expectedModel := struct{}{}

	translator.On("FromLogs", ld).Return(expectedModel, nil)
	encoder.On("EncodeLogs", expectedModel).Return(nil, errors.New("serialization failed"))

	_, err := lm.Marshal(ld)
	assert.Error(t, err)
	assert.EqualError(t, err, "marshal failed: serialization failed")
}

func TestLogsMarshal_Encode(t *testing.T) {
	translator := &mockTranslator{}
	encoder := &mockEncoder{}

	lm := NewLogsMarshaler(encoder, translator)
	expectedLogs := NewLogs()
	expectedBytes := []byte{1, 2, 3}
	expectedModel := struct{}{}

	translator.On("FromLogs", expectedLogs).Return(expectedModel, nil)
	encoder.On("EncodeLogs", expectedModel).Return(expectedBytes, nil)

	actualBytes, err := lm.Marshal(expectedLogs)
	assert.NoError(t, err)
	assert.Equal(t, expectedBytes, actualBytes)
}

func TestLogsUnmarshal_EncodingError(t *testing.T) {
	translator := &mockTranslator{}
	encoder := &mockEncoder{}

	lu := NewLogsUnmarshaler(encoder, translator)
	expectedBytes := []byte{1, 2, 3}
	expectedModel := struct{}{}

	encoder.On("DecodeLogs", expectedBytes).Return(expectedModel, errors.New("decode failed"))

	_, err := lu.Unmarshal(expectedBytes)
	assert.Error(t, err)
	assert.EqualError(t, err, "unmarshal failed: decode failed")
}

func TestLogsUnmarshal_TranslationError(t *testing.T) {
	translator := &mockTranslator{}
	encoder := &mockEncoder{}

	lu := NewLogsUnmarshaler(encoder, translator)
	expectedBytes := []byte{1, 2, 3}
	expectedModel := struct{}{}

	encoder.On("DecodeLogs", expectedBytes).Return(expectedModel, nil)
	translator.On("ToLogs", expectedModel).Return(NewLogs(), errors.New("translation failed"))

	_, err := lu.Unmarshal(expectedBytes)
	assert.Error(t, err)
	assert.EqualError(t, err, "converting model to pdata failed: translation failed")
}

func TestLogsUnmarshal_Decode(t *testing.T) {
	translator := &mockTranslator{}
	encoder := &mockEncoder{}

	lu := NewLogsUnmarshaler(encoder, translator)
	expectedLogs := NewLogs()
	expectedBytes := []byte{1, 2, 3}
	expectedModel := struct{}{}

	encoder.On("DecodeLogs", expectedBytes).Return(expectedModel, nil)
	translator.On("ToLogs", expectedModel).Return(expectedLogs, nil)

	actualLogs, err := lu.Unmarshal(expectedBytes)
	assert.NoError(t, err)
	assert.Equal(t, expectedLogs, actualLogs)
}

func TestLogRecordCount(t *testing.T) {
	md := NewLogs()
	assert.EqualValues(t, 0, md.LogRecordCount())

	rl := md.ResourceLogs().AppendEmpty()
	assert.EqualValues(t, 0, md.LogRecordCount())

	ill := rl.InstrumentationLibraryLogs().AppendEmpty()
	assert.EqualValues(t, 0, md.LogRecordCount())

	ill.Logs().AppendEmpty()
	assert.EqualValues(t, 1, md.LogRecordCount())

	rms := md.ResourceLogs()
	rms.Resize(3)
	rms.At(1).InstrumentationLibraryLogs().AppendEmpty()
	rms.At(2).InstrumentationLibraryLogs().AppendEmpty().Logs().Resize(5)
	// 5 + 1 (from rms.At(0) initialized first)
	assert.EqualValues(t, 6, md.LogRecordCount())
}

func TestLogRecordCountWithEmpty(t *testing.T) {
	assert.Zero(t, NewLogs().LogRecordCount())
	assert.Zero(t, Logs{orig: &otlpcollectorlog.ExportLogsServiceRequest{
		ResourceLogs: []*otlplogs.ResourceLogs{{}},
	}}.LogRecordCount())
	assert.Zero(t, Logs{orig: &otlpcollectorlog.ExportLogsServiceRequest{
		ResourceLogs: []*otlplogs.ResourceLogs{
			{
				InstrumentationLibraryLogs: []*otlplogs.InstrumentationLibraryLogs{{}},
			},
		},
	}}.LogRecordCount())
	assert.Equal(t, 1, Logs{orig: &otlpcollectorlog.ExportLogsServiceRequest{
		ResourceLogs: []*otlplogs.ResourceLogs{
			{
				InstrumentationLibraryLogs: []*otlplogs.InstrumentationLibraryLogs{
					{
						Logs: []*otlplogs.LogRecord{{}},
					},
				},
			},
		},
	}}.LogRecordCount())
}

func TestToFromLogProto(t *testing.T) {
	wrapper := internal.LogsFromOtlp(&otlpcollectorlog.ExportLogsServiceRequest{})
	ld := LogsFromInternalRep(wrapper)
	assert.EqualValues(t, NewLogs(), ld)
	assert.EqualValues(t, &otlpcollectorlog.ExportLogsServiceRequest{}, ld.orig)
}

func TestLogsToFromOtlpProtoBytes(t *testing.T) {
	send := NewLogs()
	fillTestResourceLogsSlice(send.ResourceLogs())
	bytes, err := send.ToOtlpProtoBytes()
	assert.NoError(t, err)

	recv, err := LogsFromOtlpProtoBytes(bytes)
	assert.NoError(t, err)
	assert.EqualValues(t, send, recv)
}

func TestLogsFromInvalidOtlpProtoBytes(t *testing.T) {
	_, err := LogsFromOtlpProtoBytes([]byte{0xFF})
	assert.EqualError(t, err, "unexpected EOF")
}

func TestLogsClone(t *testing.T) {
	logs := NewLogs()
	fillTestResourceLogsSlice(logs.ResourceLogs())
	assert.EqualValues(t, logs, logs.Clone())
}

func BenchmarkLogsClone(b *testing.B) {
	logs := NewLogs()
	fillTestResourceLogsSlice(logs.ResourceLogs())
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		clone := logs.Clone()
		if clone.ResourceLogs().Len() != logs.ResourceLogs().Len() {
			b.Fail()
		}
	}
}

func BenchmarkLogsToOtlp(b *testing.B) {
	traces := NewLogs()
	fillTestResourceLogsSlice(traces.ResourceLogs())
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		buf, err := traces.ToOtlpProtoBytes()
		require.NoError(b, err)
		assert.NotEqual(b, 0, len(buf))
	}
}

func BenchmarkLogsFromOtlp(b *testing.B) {
	baseLogs := NewLogs()
	fillTestResourceLogsSlice(baseLogs.ResourceLogs())
	buf, err := baseLogs.ToOtlpProtoBytes()
	require.NoError(b, err)
	assert.NotEqual(b, 0, len(buf))
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		logs, err := LogsFromOtlpProtoBytes(buf)
		require.NoError(b, err)
		assert.Equal(b, baseLogs.ResourceLogs().Len(), logs.ResourceLogs().Len())
	}
}
