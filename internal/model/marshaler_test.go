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

func TestTracesEncoder_TranslationError(t *testing.T) {
	translate := &mockTranslator{}
	serialize := &mockSerializer{}

	d := &TracesMarshaler{
		translate: translate,
		serialize: serialize,
	}

	td := pdata.NewTraces()

	translate.On("EncodeTraces", td).Return(nil, errors.New("translation failed"))

	_, err := d.Marshal(td)

	assert.Error(t, err)
	assert.EqualError(t, err, "converting pdata to model failed: translation failed")
}

func TestTracesEncoder_SerializeError(t *testing.T) {
	translate := &mockTranslator{}
	serialize := &mockSerializer{}

	d := &TracesMarshaler{
		translate: translate,
		serialize: serialize,
	}

	td := pdata.NewTraces()
	expectedModel := struct{}{}

	translate.On("EncodeTraces", td).Return(expectedModel, nil)
	serialize.On("MarshalTraces", expectedModel).Return(nil, errors.New("serialization failed"))

	_, err := d.Marshal(td)

	assert.Error(t, err)
	assert.EqualError(t, err, "marshal failed: serialization failed")
}

func TestTracesEncoder_Encode(t *testing.T) {
	translate := &mockTranslator{}
	serialize := &mockSerializer{}

	d := &TracesMarshaler{
		translate: translate,
		serialize: serialize,
	}

	expectedTraces := pdata.NewTraces()
	expectedBytes := []byte{1, 2, 3}
	expectedModel := struct{}{}

	translate.On("EncodeTraces", expectedTraces).Return(expectedModel, nil)
	serialize.On("MarshalTraces", expectedModel).Return(expectedBytes, nil)

	actualBytes, err := d.Marshal(expectedTraces)

	assert.NoError(t, err)
	assert.Equal(t, expectedBytes, actualBytes)
}
