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

package kafkareceiver

import (
	"testing"

	"go.opentelemetry.io/collector/receiver/otlpreceiver"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
	otlptrace "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/collector/trace/v1"
)

func TestUnmarshallOTLP(t *testing.T) {
	td := pdata.NewTraces()
	td.ResourceSpans().Resize(1)
	td.ResourceSpans().At(0).Resource().Attributes().InsertString("foo", "bar")
	request := &otlptrace.ExportTraceServiceRequest{
		ResourceSpans: pdata.TracesToOtlp(td),
	}
	expected, err := request.Marshal()
	require.NoError(t, err)

	p := otlpProtoUnmarshaller{}
	got, err := p.Unmarshal(expected)
	require.NoError(t, err)
	assert.Equal(t, td, got)
	assert.Equal(t, "otlp_proto", p.Encoding())
}

func TestUnmarshallOTLP_error(t *testing.T) {
	p := otlpProtoUnmarshaller{}
	got, err := p.Unmarshal([]byte("+$%"))
	assert.Equal(t, pdata.NewTraces(), got)
	assert.Error(t, err)
}

func TestUnmarshallJSON(t *testing.T) {
	jsonMarshaller := otlpreceiver.JSONPb{
		EmitDefaults: false,
		Indent:       "  ",
		OrigName:     true,
	}
	td := pdata.NewTraces()
	td.ResourceSpans().Resize(1)
	td.ResourceSpans().At(0).Resource().Attributes().InsertString("foo", "bar")
	request := &otlptrace.ExportTraceServiceRequest{
		ResourceSpans: pdata.TracesToOtlp(td),
	}
	data, err := jsonMarshaller.Marshal(request)
	require.NoError(t, err)

	p := newOTLPJSONUnmarshaller()
	got, err := p.Unmarshal(data)
	require.NoError(t, err)
	assert.Equal(t, td, got)
	assert.Equal(t, "otlp_json", p.Encoding())
}

func TestUnmarshallJSON_error(t *testing.T) {
	p := newOTLPJSONUnmarshaller()
	got, err := p.Unmarshal([]byte("+$%"))
	assert.Equal(t, pdata.NewTraces(), got)
	assert.Error(t, err)
}
