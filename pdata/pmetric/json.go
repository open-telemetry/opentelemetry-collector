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

package pmetric // import "go.opentelemetry.io/collector/pdata/pmetric"

import (
	"bytes"

	"github.com/gogo/protobuf/jsonpb"

	"go.opentelemetry.io/collector/pdata/internal"
	otlpmetrics "go.opentelemetry.io/collector/pdata/internal/data/protogen/metrics/v1"
	"go.opentelemetry.io/collector/pdata/pmetric/internal/pmetricjson"
)

// NewJSONMarshaler returns a model.Marshaler. Marshals to OTLP json bytes.
func NewJSONMarshaler() Marshaler {
	return &jsonMarshaler{delegate: jsonpb.Marshaler{
		EnumsAsInts: true,
	}}
}

type jsonMarshaler struct {
	delegate jsonpb.Marshaler
}

func (e *jsonMarshaler) MarshalMetrics(md Metrics) ([]byte, error) {
	buf := bytes.Buffer{}
	pb := internal.MetricsToProto(internal.Metrics(md))
	err := e.delegate.Marshal(&buf, &pb)
	return buf.Bytes(), err
}

type jsonUnmarshaler struct{}

// NewJSONUnmarshaler returns a model.Unmarshaler. Unmarshals from OTLP json bytes.
func NewJSONUnmarshaler() Unmarshaler {
	return &jsonUnmarshaler{}
}

func (jsonUnmarshaler) UnmarshalMetrics(buf []byte) (Metrics, error) {
	var md otlpmetrics.MetricsData
	if err := pmetricjson.UnmarshalMetricsData(buf, &md); err != nil {
		return Metrics{}, err
	}
	return Metrics(internal.MetricsFromProto(md)), nil
}
