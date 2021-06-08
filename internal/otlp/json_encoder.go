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

package otlp

import (
	"bytes"

	"github.com/gogo/protobuf/jsonpb"

	otlpcollectorlogs "go.opentelemetry.io/collector/internal/data/protogen/collector/logs/v1"
	otlpcollectormetrics "go.opentelemetry.io/collector/internal/data/protogen/collector/metrics/v1"
	otlpcollectortrace "go.opentelemetry.io/collector/internal/data/protogen/collector/trace/v1"
	"go.opentelemetry.io/collector/internal/model"
)

type jsonEncoder struct {
	delegate jsonpb.Marshaler
}

func newJSONEncoder() *jsonEncoder {
	return &jsonEncoder{delegate: jsonpb.Marshaler{}}
}

func (e *jsonEncoder) EncodeLogs(modelData interface{}) ([]byte, error) {
	ld, ok := modelData.(*otlpcollectorlogs.ExportLogsServiceRequest)
	if !ok {
		return nil, model.NewErrIncompatibleType(&otlpcollectorlogs.ExportLogsServiceRequest{}, modelData)
	}
	buf := bytes.Buffer{}
	if err := e.delegate.Marshal(&buf, ld); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (e *jsonEncoder) EncodeMetrics(modelData interface{}) ([]byte, error) {
	md, ok := modelData.(*otlpcollectormetrics.ExportMetricsServiceRequest)
	if !ok {
		return nil, model.NewErrIncompatibleType(&otlpcollectormetrics.ExportMetricsServiceRequest{}, modelData)
	}
	buf := bytes.Buffer{}
	if err := e.delegate.Marshal(&buf, md); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (e *jsonEncoder) EncodeTraces(modelData interface{}) ([]byte, error) {
	td, ok := modelData.(*otlpcollectortrace.ExportTraceServiceRequest)
	if !ok {
		return nil, model.NewErrIncompatibleType(&otlpcollectortrace.ExportTraceServiceRequest{}, modelData)
	}
	buf := bytes.Buffer{}
	if err := e.delegate.Marshal(&buf, td); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
