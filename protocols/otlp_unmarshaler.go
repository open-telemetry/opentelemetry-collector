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

package protocols

import (
	"go.opentelemetry.io/collector/consumer/pdata"
)

// OTLPProtobuf protocol type.
const OTLPProtobuf Type = "otlp/protobuf"

type otlpTracesPbUnmarshaler struct {
}

var _ TracesUnmarshaler = (*otlpTracesPbUnmarshaler)(nil)

func (p *otlpTracesPbUnmarshaler) Unmarshal(bytes []byte) (pdata.Traces, error) {
	return pdata.TracesFromOtlpProtoBytes(bytes)
}

func (*otlpTracesPbUnmarshaler) Type() Type {
	return OTLPProtobuf
}

type otlpLogsPbUnmarshaler struct {
}

var _ LogsUnmarshaler = (*otlpLogsPbUnmarshaler)(nil)

func (p *otlpLogsPbUnmarshaler) Unmarshal(bytes []byte) (pdata.Logs, error) {
	return pdata.LogsFromOtlpProtoBytes(bytes)
}

func (*otlpLogsPbUnmarshaler) Type() Type {
	return OTLPProtobuf
}
