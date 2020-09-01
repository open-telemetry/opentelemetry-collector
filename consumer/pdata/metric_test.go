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

package pdata

import (
	"testing"

	gogoproto "github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
	goproto "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"

	otlpmetrics "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/metrics/v1"
)

func TestCopyData(t *testing.T) {
	tests := []struct {
		name string
		src  *otlpmetrics.Metric
	}{
		{
			name: "IntGauge",
			src: &otlpmetrics.Metric{
				Data: &otlpmetrics.Metric_IntGauge{
					IntGauge: &otlpmetrics.IntGauge{},
				},
			},
		},
		{
			name: "DoubleGauge",
			src: &otlpmetrics.Metric{
				Data: &otlpmetrics.Metric_DoubleGauge{
					DoubleGauge: &otlpmetrics.DoubleGauge{},
				},
			},
		},
		{
			name: "IntSum",
			src: &otlpmetrics.Metric{
				Data: &otlpmetrics.Metric_IntSum{
					IntSum: &otlpmetrics.IntSum{},
				},
			},
		},
		{
			name: "DoubleSum",
			src: &otlpmetrics.Metric{
				Data: &otlpmetrics.Metric_DoubleSum{
					DoubleSum: &otlpmetrics.DoubleSum{},
				},
			},
		},
		{
			name: "IntHistogram",
			src: &otlpmetrics.Metric{
				Data: &otlpmetrics.Metric_IntHistogram{
					IntHistogram: &otlpmetrics.IntHistogram{},
				},
			},
		},
		{
			name: "DoubleHistogram",
			src: &otlpmetrics.Metric{
				Data: &otlpmetrics.Metric_DoubleHistogram{
					DoubleHistogram: &otlpmetrics.DoubleHistogram{},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dest := &otlpmetrics.Metric{}
			assert.Nil(t, dest.Data)
			assert.NotNil(t, test.src.Data)
			copyData(test.src, dest)
			assert.EqualValues(t, test.src, dest)
		})
	}
}

func TestDataType(t *testing.T) {
	m := NewMetric()
	m.InitEmpty()
	assert.Equal(t, MetricDataTypeNone, m.DataType())
	m.SetDataType(MetricDataTypeIntGauge)
	assert.Equal(t, MetricDataTypeIntGauge, m.DataType())
	assert.True(t, m.IntGauge().IsNil())
	m.SetDataType(MetricDataTypeDoubleGauge)
	assert.Equal(t, MetricDataTypeDoubleGauge, m.DataType())
	assert.True(t, m.DoubleGauge().IsNil())
	m.SetDataType(MetricDataTypeIntSum)
	assert.Equal(t, MetricDataTypeIntSum, m.DataType())
	assert.True(t, m.IntSum().IsNil())
	m.SetDataType(MetricDataTypeDoubleSum)
	assert.Equal(t, MetricDataTypeDoubleSum, m.DataType())
	assert.True(t, m.DoubleSum().IsNil())
	m.SetDataType(MetricDataTypeIntHistogram)
	assert.Equal(t, MetricDataTypeIntHistogram, m.DataType())
	assert.True(t, m.IntHistogram().IsNil())
	m.SetDataType(MetricDataTypeDoubleHistogram)
	assert.Equal(t, MetricDataTypeDoubleHistogram, m.DataType())
	assert.True(t, m.DoubleHistogram().IsNil())
	m.InitEmpty()
	assert.Equal(t, MetricDataTypeNone, m.DataType())
}

func TestResourceMetricsWireCompatibility(t *testing.T) {
	// This test verifies that OTLP ProtoBufs generated using goproto lib in
	// opentelemetry-proto repository OTLP ProtoBufs generated using gogoproto lib in
	// this repository are wire compatible.

	// Generate ResourceMetrics as pdata struct.
	pdataRM := generateTestResourceMetrics()

	// Marshal its underlying ProtoBuf to wire.
	wire1, err := gogoproto.Marshal(*pdataRM.orig)
	assert.NoError(t, err)
	assert.NotNil(t, wire1)

	// Unmarshal from the wire to OTLP Protobuf in goproto's representation.
	var goprotoMessage emptypb.Empty
	err = goproto.Unmarshal(wire1, &goprotoMessage)
	assert.NoError(t, err)

	// Marshal to the wire again.
	wire2, err := goproto.Marshal(&goprotoMessage)
	assert.NoError(t, err)
	assert.NotNil(t, wire2)

	// Unmarshal from the wire into gogoproto's representation.
	var gogoprotoRM otlpmetrics.ResourceMetrics
	err = gogoproto.Unmarshal(wire2, &gogoprotoRM)
	assert.NoError(t, err)

	// Now compare that the original and final ProtoBuf messages are the same.
	// This proves that goproto and gogoproto marshaling/unmarshaling are wire compatible.
	assert.True(t, gogoproto.Equal(*pdataRM.orig, &gogoprotoRM))
}
