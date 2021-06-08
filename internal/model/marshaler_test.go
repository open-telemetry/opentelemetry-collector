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

package model

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/consumer/pdata"
)

func TestTracesMarshal_TranslationError(t *testing.T) {
	translator := &mockTranslator{}
	encoder := &mockEncoder{}

	tm := NewTracesMarshaler(encoder, translator)
	td := pdata.NewTraces()

	translator.On("FromTraces", td).Return(nil, errors.New("translation failed"))

	_, err := tm.Marshal(td)
	assert.Error(t, err)
	assert.EqualError(t, err, "converting pdata to model failed: translation failed")
}

func TestTracesMarshal_SerializeError(t *testing.T) {
	translator := &mockTranslator{}
	encoder := &mockEncoder{}

	tm := NewTracesMarshaler(encoder, translator)
	td := pdata.NewTraces()
	expectedModel := struct{}{}

	translator.On("FromTraces", td).Return(expectedModel, nil)
	encoder.On("EncodeTraces", expectedModel).Return(nil, errors.New("serialization failed"))

	_, err := tm.Marshal(td)
	assert.Error(t, err)
	assert.EqualError(t, err, "marshal failed: serialization failed")
}

func TestTracesMarshal_Encode(t *testing.T) {
	translator := &mockTranslator{}
	encoder := &mockEncoder{}

	tm := NewTracesMarshaler(encoder, translator)
	expectedTraces := pdata.NewTraces()
	expectedBytes := []byte{1, 2, 3}
	expectedModel := struct{}{}

	translator.On("FromTraces", expectedTraces).Return(expectedModel, nil)
	encoder.On("EncodeTraces", expectedModel).Return(expectedBytes, nil)

	actualBytes, err := tm.Marshal(expectedTraces)
	assert.NoError(t, err)
	assert.Equal(t, expectedBytes, actualBytes)
}

func TestMetricsMarshal_TranslationError(t *testing.T) {
	translator := &mockTranslator{}
	encoder := &mockEncoder{}

	mm := NewMetricsMarshaler(encoder, translator)
	md := pdata.NewMetrics()

	translator.On("FromMetrics", md).Return(nil, errors.New("translation failed"))

	_, err := mm.Marshal(md)
	assert.Error(t, err)
	assert.EqualError(t, err, "converting pdata to model failed: translation failed")
}

func TestMetricsMarshal_SerializeError(t *testing.T) {
	translator := &mockTranslator{}
	encoder := &mockEncoder{}

	mm := NewMetricsMarshaler(encoder, translator)
	md := pdata.NewMetrics()
	expectedModel := struct{}{}

	translator.On("FromMetrics", md).Return(expectedModel, nil)
	encoder.On("EncodeMetrics", expectedModel).Return(nil, errors.New("serialization failed"))

	_, err := mm.Marshal(md)
	assert.Error(t, err)
	assert.EqualError(t, err, "marshal failed: serialization failed")
}

func TestMetricsMarshal_Encode(t *testing.T) {
	translator := &mockTranslator{}
	encoder := &mockEncoder{}

	mm := NewMetricsMarshaler(encoder, translator)
	expectedMetrics := pdata.NewMetrics()
	expectedBytes := []byte{1, 2, 3}
	expectedModel := struct{}{}

	translator.On("FromMetrics", expectedMetrics).Return(expectedModel, nil)
	encoder.On("EncodeMetrics", expectedModel).Return(expectedBytes, nil)

	actualBytes, err := mm.Marshal(expectedMetrics)
	assert.NoError(t, err)
	assert.Equal(t, expectedBytes, actualBytes)
}

func TestLogsMarshal_TranslationError(t *testing.T) {
	translator := &mockTranslator{}
	encoder := &mockEncoder{}

	lm := NewLogsMarshaler(encoder, translator)
	ld := pdata.NewLogs()

	translator.On("FromLogs", ld).Return(nil, errors.New("translation failed"))

	_, err := lm.Marshal(ld)
	assert.Error(t, err)
	assert.EqualError(t, err, "converting pdata to model failed: translation failed")
}

func TestLogsMarshal_SerializeError(t *testing.T) {
	translator := &mockTranslator{}
	encoder := &mockEncoder{}

	lm := NewLogsMarshaler(encoder, translator)
	ld := pdata.NewLogs()
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
	expectedLogs := pdata.NewLogs()
	expectedBytes := []byte{1, 2, 3}
	expectedModel := struct{}{}

	translator.On("FromLogs", expectedLogs).Return(expectedModel, nil)
	encoder.On("EncodeLogs", expectedModel).Return(expectedBytes, nil)

	actualBytes, err := lm.Marshal(expectedLogs)
	assert.NoError(t, err)
	assert.Equal(t, expectedBytes, actualBytes)
}
