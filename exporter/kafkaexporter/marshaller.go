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

package kafkaexporter

import (
	"go.opentelemetry.io/collector/consumer/pdata"
	otlptrace "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/collector/trace/v1"
)

// marshaller encodes traces into a byte array to be sent to Kafka.
type marshaller interface {
	// Marshal serializes spans into byte array
	Marshal(traces pdata.Traces) ([]byte, error)
}

type protoMarshaller struct {
}

func (m *protoMarshaller) Marshal(traces pdata.Traces) ([]byte, error) {
	request := otlptrace.ExportTraceServiceRequest{
		ResourceSpans: pdata.TracesToOtlp(traces),
	}
	return request.Marshal()
}
