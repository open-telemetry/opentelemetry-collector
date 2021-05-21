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

func TestTracesDecoder_SerializeError(t *testing.T) {
	translate := &mockTranslator{}
	serialize := &mockSerializer{}

	d := &TracesDecoder{
		translate: translate,
		serialize: serialize,
	}

	expectedBytes := []byte{1, 2, 3}
	expectedModel := struct{}{}

	serialize.On("UnmarshalTraces", expectedBytes).Return(expectedModel, errors.New("decode failed"))

	_, err := d.Decode(expectedBytes)

	assert.Error(t, err)
	assert.EqualError(t, err, "unmarshal failed: decode failed")
}

func TestTracesDecoder_TranslationError(t *testing.T) {
	translate := &mockTranslator{}
	serialize := &mockSerializer{}

	d := &TracesDecoder{
		translate: translate,
		serialize: serialize,
	}

	expectedBytes := []byte{1, 2, 3}
	expectedModel := struct{}{}

	serialize.On("UnmarshalTraces", expectedBytes).Return(expectedModel, nil)
	translate.On("DecodeTraces", expectedModel).Return(pdata.NewTraces(), errors.New("translation failed"))

	_, err := d.Decode(expectedBytes)

	assert.Error(t, err)
	assert.EqualError(t, err, "converting model to pdata failed: translation failed")
}

func TestTracesDecoder_Decode(t *testing.T) {
	translate := &mockTranslator{}
	serialize := &mockSerializer{}

	d := &TracesDecoder{
		translate: translate,
		serialize: serialize,
	}

	expectedTraces := pdata.NewTraces()
	expectedBytes := []byte{1, 2, 3}
	expectedModel := struct{}{}

	serialize.On("UnmarshalTraces", expectedBytes).Return(expectedModel, nil)
	translate.On("DecodeTraces", expectedModel).Return(expectedTraces, nil)

	actualTraces, err := d.Decode(expectedBytes)

	assert.NoError(t, err)
	assert.Equal(t, expectedTraces, actualTraces)
}
