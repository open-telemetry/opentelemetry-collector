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
)

type jsonDecoder struct {
	delegate jsonpb.Unmarshaler
}

func newJSONDecoder() *jsonDecoder {
	return &jsonDecoder{delegate: jsonpb.Unmarshaler{}}
}

func (d *jsonDecoder) DecodeLogs(buf []byte) (interface{}, error) {
	ld := &otlpcollectorlogs.ExportLogsServiceRequest{}
	err := d.delegate.Unmarshal(bytes.NewReader(buf), ld)
	return ld, err
}

func (d *jsonDecoder) DecodeMetrics(buf []byte) (interface{}, error) {
	md := &otlpcollectormetrics.ExportMetricsServiceRequest{}
	err := d.delegate.Unmarshal(bytes.NewReader(buf), md)
	return md, err
}

func (d *jsonDecoder) DecodeTraces(buf []byte) (interface{}, error) {
	td := &otlpcollectortrace.ExportTraceServiceRequest{}
	err := d.delegate.Unmarshal(bytes.NewReader(buf), td)
	return td, err
}
