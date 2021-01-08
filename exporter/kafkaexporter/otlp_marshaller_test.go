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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
	otlpmetric "go.opentelemetry.io/collector/internal/data/protogen/collector/metrics/v1"
	otlptrace "go.opentelemetry.io/collector/internal/data/protogen/collector/trace/v1"
	"go.opentelemetry.io/collector/internal/testdata"
)

func TestOTLPTracesPbMarshaller(t *testing.T) {
	td := testdata.GenerateTraceDataTwoSpansSameResource()
	request := &otlptrace.ExportTraceServiceRequest{
		ResourceSpans: pdata.TracesToOtlp(td),
	}
	expected, err := request.Marshal()
	require.NoError(t, err)
	require.NotNil(t, expected)

	m := otlpTracesPbMarshaller{}
	assert.Equal(t, "otlp_proto", m.Encoding())
	messages, err := m.Marshal(td)
	require.NoError(t, err)
	assert.Equal(t, []Message{{Value: expected}}, messages)
}

func TestOTLPMetricsPbMarshaller(t *testing.T) {
	md := testdata.GenerateMetricsTwoMetrics()
	request := &otlpmetric.ExportMetricsServiceRequest{
		ResourceMetrics: pdata.MetricsToOtlp(md),
	}
	expected, err := request.Marshal()
	require.NoError(t, err)
	require.NotNil(t, expected)

	m := otlpMetricsPbMarshaller{}
	assert.Equal(t, "otlp_proto", m.Encoding())
	messages, err := m.Marshal(md)
	require.NoError(t, err)
	assert.Equal(t, []Message{{Value: expected}}, messages)
}
