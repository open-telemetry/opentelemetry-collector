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
	"go.opentelemetry.io/collector/internal/testdata"
)

func TestOTLPTracesPbMarshaller(t *testing.T) {
	td := testdata.GenerateTraceDataTwoSpansSameResource()
	m := otlpTracesPbMarshaller{}
	assert.Equal(t, "otlp_proto", m.Encoding())
	messages, err := m.Marshal(td)
	require.NoError(t, err)
	require.Len(t, messages, 1)
	extracted := pdata.NewTraces()
	err = extracted.FromOtlpProtoBytes(messages[0].Value)
	require.NoError(t, err)
	assert.EqualValues(t, td, extracted)
}

func TestOTLPMetricsPbMarshaller(t *testing.T) {
	md := testdata.GenerateMetricsTwoMetrics()
	m := otlpMetricsPbMarshaller{}
	assert.Equal(t, "otlp_proto", m.Encoding())
	messages, err := m.Marshal(md)
	require.NoError(t, err)
	require.Len(t, messages, 1)
	extracted, err := pdata.MetricsFromOtlpProtoBytes(messages[0].Value)
	require.NoError(t, err)
	assert.EqualValues(t, md, extracted)
}
