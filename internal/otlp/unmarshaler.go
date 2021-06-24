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
	"go.opentelemetry.io/collector/model/pdata"
)

// NewJSONTracesUnmarshaler returns a model.TracesUnmarshaler. Unmarshals from OTLP json bytes.
func NewJSONTracesUnmarshaler() pdata.TracesUnmarshaler {
	return pdata.NewTracesUnmarshaler(newJSONDecoder(), newToTranslator())
}

// NewJSONMetricsUnmarshaler returns a model.MetricsUnmarshaler. Unmarshals from OTLP json bytes.
func NewJSONMetricsUnmarshaler() pdata.MetricsUnmarshaler {
	return pdata.NewMetricsUnmarshaler(newJSONDecoder(), newToTranslator())
}

// NewJSONLogsUnmarshaler returns a model.LogsUnmarshaler. Unmarshals from OTLP json bytes.
func NewJSONLogsUnmarshaler() pdata.LogsUnmarshaler {
	return pdata.NewLogsUnmarshaler(newJSONDecoder(), newToTranslator())
}

// NewProtobufTracesUnmarshaler returns a model.TracesUnmarshaler. Unmarshals from OTLP binary protobuf bytes.
func NewProtobufTracesUnmarshaler() pdata.TracesUnmarshaler {
	return pdata.NewTracesUnmarshaler(newPbDecoder(), newToTranslator())
}

// NewProtobufMetricsUnmarshaler returns a model.MetricsUnmarshaler. Unmarshals from OTLP binary protobuf bytes.
func NewProtobufMetricsUnmarshaler() pdata.MetricsUnmarshaler {
	return pdata.NewMetricsUnmarshaler(newPbDecoder(), newToTranslator())
}

// NewProtobufLogsUnmarshaler returns a model.LogsUnmarshaler. Unmarshals from OTLP binary protobuf bytes.
func NewProtobufLogsUnmarshaler() pdata.LogsUnmarshaler {
	return pdata.NewLogsUnmarshaler(newPbDecoder(), newToTranslator())
}
