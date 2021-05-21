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
	"github.com/stretchr/testify/mock"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/model/serializer"
	"go.opentelemetry.io/collector/internal/model/translator"
)

var (
	_ serializer.TracesMarshaler   = (*mockSerializer)(nil)
	_ serializer.TracesUnmarshaler = (*mockSerializer)(nil)
)

type mockSerializer struct {
	mock.Mock
}

func (m *mockSerializer) MarshalTraces(model interface{}) ([]byte, error) {
	args := m.Called(model)
	err := args.Error(1)
	if err != nil {
		return nil, err
	}
	return args.Get(0).([]byte), err
}

func (m *mockSerializer) UnmarshalTraces(bytes []byte) (interface{}, error) {
	args := m.Called(bytes)
	return args.Get(0), args.Error(1)
}

var (
	_ translator.TracesEncoder = (*mockTranslator)(nil)
	_ translator.TracesDecoder = (*mockTranslator)(nil)
)

type mockTranslator struct {
	mock.Mock
}

func (m *mockTranslator) DecodeTraces(src interface{}) (pdata.Traces, error) {
	args := m.Called(src)
	return args.Get(0).(pdata.Traces), args.Error(1)
}

func (m *mockTranslator) EncodeTraces(md pdata.Traces) (interface{}, error) {
	args := m.Called(md)
	return args.Get(0), args.Error(1)
}
