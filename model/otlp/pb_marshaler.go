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
	"fmt"

	"go.opentelemetry.io/collector/model/internal"
	"go.opentelemetry.io/collector/model/pdata"
)

// NewProtobufTracesMarshaler returns a model.TracesMarshaler. Marshals to OTLP binary protobuf bytes.
func NewProtobufTracesMarshaler() pdata.TracesMarshaler {
	return newPbMarshaler()
}

// NewProtobufMetricsMarshaler returns a model.MetricsMarshaler. Marshals to OTLP binary protobuf bytes.
func NewProtobufMetricsMarshaler() pdata.MetricsMarshaler {
	return newPbMarshaler()
}

// NewProtobufLogsMarshaler returns a model.LogsMarshaler. Marshals to OTLP binary protobuf bytes.
func NewProtobufLogsMarshaler() pdata.LogsMarshaler {
	return newPbMarshaler()
}

func NewProtobufSizer() pdata.Sizer {
	return newPbMarshaler()
}

type pbMarshaler struct{}

func newPbMarshaler() *pbMarshaler {
	return &pbMarshaler{}
}

func (e *pbMarshaler) MarshalLogs(ld pdata.Logs) ([]byte, error) {
	return internal.LogsToOtlp(ld.InternalRep()).Marshal()
}

func (e *pbMarshaler) MarshalMetrics(md pdata.Metrics) ([]byte, error) {
	return internal.MetricsToOtlp(md.InternalRep()).Marshal()
}

func (e *pbMarshaler) MarshalTraces(td pdata.Traces) ([]byte, error) {
	return internal.TracesToOtlp(td.InternalRep()).Marshal()
}

// Size returns the size in bytes of a pdata.Traces, pdata.Metrics or pdata.Logs.
// If the type is not known, an error will be returned.
func (e *pbMarshaler) Size(v interface{}) (int, error) {
	switch conv := v.(type) {
	case pdata.Traces:
		return internal.TracesToOtlp(conv.InternalRep()).Size(), nil
	case *pdata.Traces:
		return internal.TracesToOtlp(conv.InternalRep()).Size(), nil
	case pdata.Metrics:
		return internal.MetricsToOtlp(conv.InternalRep()).Size(), nil
	case *pdata.Metrics:
		return internal.MetricsToOtlp(conv.InternalRep()).Size(), nil
	case pdata.Logs:
		return internal.LogsToOtlp(conv.InternalRep()).Size(), nil
	case *pdata.Logs:
		return internal.LogsToOtlp(conv.InternalRep()).Size(), nil
	default:
		return 0, fmt.Errorf("unknown type: %T", v)
	}
}
