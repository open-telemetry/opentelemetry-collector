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
	"go.opentelemetry.io/collector/pdata/internal"
	otlpmetrics "go.opentelemetry.io/collector/pdata/internal/data/protogen/metrics/v1"
)

// NewProtoMarshaler returns a Marshaler. Marshals to OTLP binary protobuf bytes.
func NewProtoMarshaler() Marshaler {
	return newPbMarshaler()
}

// TODO(#3842): Figure out how we want to represent/return *Sizers.
type pbMarshaler struct{}

func newPbMarshaler() *pbMarshaler {
	return &pbMarshaler{}
}

var _ Sizer = (*pbMarshaler)(nil)

func (e *pbMarshaler) MarshalMetrics(md Metrics) ([]byte, error) {
	return internal.MetricsToOtlp(md).Marshal()
}

func (e *pbMarshaler) MetricsSize(md Metrics) int {
	return internal.MetricsToOtlp(md).Size()
}

type pbUnmarshaler struct{}

// NewProtoUnmarshaler returns a model.Unmarshaler. Unmarshals from OTLP binary protobuf bytes.
func NewProtoUnmarshaler() Unmarshaler {
	return newPbUnmarshaler()
}

func newPbUnmarshaler() *pbUnmarshaler {
	return &pbUnmarshaler{}
}

func (d *pbUnmarshaler) UnmarshalMetrics(buf []byte) (Metrics, error) {
	md := &otlpmetrics.MetricsData{}
	err := md.Unmarshal(buf)
	return internal.MetricsFromOtlp(md), err
}
