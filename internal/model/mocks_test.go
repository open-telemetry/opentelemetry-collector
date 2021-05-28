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
)

var (
	_ TracesEncoder = (*mockEncoder)(nil)
	_ TracesDecoder = (*mockEncoder)(nil)
)

type mockEncoder struct {
	mock.Mock
}

func (m *mockEncoder) EncodeTraces(model interface{}) ([]byte, error) {
	args := m.Called(model)
	err := args.Error(1)
	if err != nil {
		return nil, err
	}
	return args.Get(0).([]byte), err
}

func (m *mockEncoder) DecodeTraces(bytes []byte) (interface{}, error) {
	args := m.Called(bytes)
	return args.Get(0), args.Error(1)
}

func (m *mockEncoder) EncodeMetrics(model interface{}) ([]byte, error) {
	args := m.Called(model)
	err := args.Error(1)
	if err != nil {
		return nil, err
	}
	return args.Get(0).([]byte), err
}

func (m *mockEncoder) DecodeMetrics(bytes []byte) (interface{}, error) {
	args := m.Called(bytes)
	return args.Get(0), args.Error(1)
}

func (m *mockEncoder) EncodeLogs(model interface{}) ([]byte, error) {
	args := m.Called(model)
	err := args.Error(1)
	if err != nil {
		return nil, err
	}
	return args.Get(0).([]byte), err
}

func (m *mockEncoder) DecodeLogs(bytes []byte) (interface{}, error) {
	args := m.Called(bytes)
	return args.Get(0), args.Error(1)
}

var (
	_ ToTracesTranslator   = (*mockTranslator)(nil)
	_ FromTracesTranslator = (*mockTranslator)(nil)
)

type mockTranslator struct {
	mock.Mock
}

func (m *mockTranslator) ToTraces(src interface{}) (pdata.Traces, error) {
	args := m.Called(src)
	return args.Get(0).(pdata.Traces), args.Error(1)
}

func (m *mockTranslator) FromTraces(md pdata.Traces) (interface{}, error) {
	args := m.Called(md)
	return args.Get(0), args.Error(1)
}

func (m *mockTranslator) ToMetrics(src interface{}) (pdata.Metrics, error) {
	args := m.Called(src)
	return args.Get(0).(pdata.Metrics), args.Error(1)
}

func (m *mockTranslator) FromMetrics(md pdata.Metrics) (interface{}, error) {
	args := m.Called(md)
	return args.Get(0), args.Error(1)
}

func (m *mockTranslator) ToLogs(src interface{}) (pdata.Logs, error) {
	args := m.Called(src)
	return args.Get(0).(pdata.Logs), args.Error(1)
}

func (m *mockTranslator) FromLogs(md pdata.Logs) (interface{}, error) {
	args := m.Called(md)
	return args.Get(0), args.Error(1)
}
