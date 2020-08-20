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

package kafkareceiver

import (
	"go.opentelemetry.io/collector/consumer/pdata"
	otlptrace "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/collector/trace/v1"
)

// unmarshaller deserializes the message body.
type unmarshaller interface {
	// Unmarshal deserializes the message body into traces.
	Unmarshal([]byte) (pdata.Traces, error)

	// Format of the encoding.
	Format() string
}

type protoUnmarshaller struct {
}

func (p *protoUnmarshaller) Unmarshal(bytes []byte) (pdata.Traces, error) {
	request := &otlptrace.ExportTraceServiceRequest{}
	err := request.Unmarshal(bytes)
	if err != nil {
		return pdata.NewTraces(), err
	}
	return pdata.TracesFromOtlp(request.GetResourceSpans()), nil
}

func (*protoUnmarshaller) Format() string {
	return "otlp_proto"
}
