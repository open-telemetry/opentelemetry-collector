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
	"go.opentelemetry.io/collector/internal/model"
)

// NewJSONTracesMarshaler returns a model.TracesMarshaler. Marshals to OTLP json bytes.
func NewJSONTracesMarshaler() model.TracesMarshaler {
	return model.NewTracesMarshaler(newJSONEncoder(), newFromTranslator())
}

// NewJSONMetricsMarshaler returns a model.MetricsMarshaler. Marshals to OTLP json bytes.
func NewJSONMetricsMarshaler() model.MetricsMarshaler {
	return model.NewMetricsMarshaler(newJSONEncoder(), newFromTranslator())
}

// NewJSONLogsMarshaler returns a model.LogsMarshaler. Marshals to OTLP json bytes.
func NewJSONLogsMarshaler() model.LogsMarshaler {
	return model.NewLogsMarshaler(newJSONEncoder(), newFromTranslator())
}

// NewProtobufTracesMarshaler returns a model.TracesMarshaler. Marshals to OTLP binary protobuf bytes.
func NewProtobufTracesMarshaler() model.TracesMarshaler {
	return model.NewTracesMarshaler(newPbEncoder(), newFromTranslator())
}

// NewProtobufMetricsMarshaler returns a model.MetricsMarshaler. Marshals to OTLP binary protobuf bytes.
func NewProtobufMetricsMarshaler() model.MetricsMarshaler {
	return model.NewMetricsMarshaler(newPbEncoder(), newFromTranslator())
}

// NewProtobufLogsMarshaler returns a model.LogsMarshaler. Marshals to OTLP binary protobuf bytes.
func NewProtobufLogsMarshaler() model.LogsMarshaler {
	return model.NewLogsMarshaler(newPbEncoder(), newFromTranslator())
}
