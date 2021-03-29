// Copyright 2020 The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kafkaexporter

import (
	"go.opentelemetry.io/collector/consumer/pdata"
)

var _ TracesMarshaller = (*otlpTracesPbMarshaller)(nil)
var _ MetricsMarshaller = (*otlpMetricsPbMarshaller)(nil)

type otlpTracesPbMarshaller struct {
}

func (m *otlpTracesPbMarshaller) Encoding() string {
	return defaultEncoding
}

func (m *otlpTracesPbMarshaller) Marshal(td pdata.Traces) ([]Message, error) {
	bts, err := td.ToOtlpProtoBytes()
	if err != nil {
		return nil, err
	}
	return []Message{{Value: bts}}, nil
}

type otlpMetricsPbMarshaller struct {
}

func (m *otlpMetricsPbMarshaller) Encoding() string {
	return defaultEncoding
}

func (m *otlpMetricsPbMarshaller) Marshal(md pdata.Metrics) ([]Message, error) {
	bts, err := md.ToOtlpProtoBytes()
	if err != nil {
		return nil, err
	}
	return []Message{{Value: bts}}, nil
}
