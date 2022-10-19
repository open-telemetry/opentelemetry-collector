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

package pmetricotlp // import "go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"

import (
	"bytes"

	"go.opentelemetry.io/collector/pdata/internal"
	otlpcollectormetrics "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/metrics/v1"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pmetric/internal/pmetricjson"
)

// Request represents the request for gRPC/HTTP client/server.
// It's a wrapper for pmetric.Metrics data.
type Request struct {
	orig *otlpcollectormetrics.ExportMetricsServiceRequest
}

// NewRequest returns an empty Request.
func NewRequest() Request {
	return Request{orig: &otlpcollectormetrics.ExportMetricsServiceRequest{}}
}

// NewRequestFromMetrics returns a Request from pmetric.Metrics.
// Because Request is a wrapper for pmetric.Metrics,
// any changes to the provided Metrics struct will be reflected in the Request and vice versa.
func NewRequestFromMetrics(md pmetric.Metrics) Request {
	return Request{orig: internal.GetOrigMetrics(internal.Metrics(md))}
}

// MarshalProto marshals Request into proto bytes.
func (mr Request) MarshalProto() ([]byte, error) {
	return mr.orig.Marshal()
}

// UnmarshalProto unmarshalls Request from proto bytes.
func (mr Request) UnmarshalProto(data []byte) error {
	return mr.orig.Unmarshal(data)
}

// MarshalJSON marshals Request into JSON bytes.
func (mr Request) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	if err := pmetricjson.JSONMarshaler.Marshal(&buf, mr.orig); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// UnmarshalJSON unmarshalls Request from JSON bytes.
func (mr Request) UnmarshalJSON(data []byte) error {
	return pmetricjson.UnmarshalExportMetricsServiceRequest(data, mr.orig)
}

func (mr Request) Metrics() pmetric.Metrics {
	return pmetric.Metrics(internal.NewMetrics(mr.orig))
}
