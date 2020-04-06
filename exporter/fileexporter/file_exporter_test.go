// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package fileexporter

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
)

type mockFile struct {
	buf bytes.Buffer
}

func (mf *mockFile) Write(p []byte) (n int, err error) {
	return mf.buf.Write(p)
}

func (mf *mockFile) Close() error {
	return nil
}

func TestFileTraceExporterNoErrors(t *testing.T) {
	mf := &mockFile{}
	lte := &Exporter{file: mf}
	require.NotNil(t, lte)

	td := consumerdata.TraceData{
		Resource: &resourcepb.Resource{
			Type:   "ServiceA",
			Labels: map[string]string{"attr1": "value1"},
		},
		Spans: []*tracepb.Span{
			{
				TraceId: []byte("123"),
				SpanId:  []byte("456"),
				Name:    &tracepb.TruncatableString{Value: "Checkout"},
				Kind:    tracepb.Span_CLIENT,
			},
			{
				Name: &tracepb.TruncatableString{Value: "Frontend"},
				Kind: tracepb.Span_SERVER,
			},
		},
	}
	assert.NoError(t, lte.ConsumeTraceData(context.Background(), td))
	assert.NoError(t, lte.Shutdown(context.Background()))

	var j map[string]interface{}
	assert.NoError(t, json.Unmarshal(mf.buf.Bytes(), &j))

	assert.EqualValues(t, j,
		map[string]interface{}{
			"resource": map[string]interface{}{
				"type":   "ServiceA",
				"labels": map[string]interface{}{"attr1": "value1"},
			},
			"spans": []interface{}{
				map[string]interface{}{
					"traceId": "MTIz", // base64 encoding of "123"
					"spanId":  "NDU2", // base64 encoding of "456"
					"kind":    "CLIENT",
					"name":    map[string]interface{}{"value": "Checkout"},
				},
				map[string]interface{}{
					"kind": "SERVER",
					"name": map[string]interface{}{"value": "Frontend"},
				},
			},
		})
}

func TestFileMetricsExporterNoErrors(t *testing.T) {
	mf := &mockFile{}
	lme := &Exporter{file: mf}
	require.NotNil(t, lme)

	md := consumerdata.MetricsData{
		Resource: &resourcepb.Resource{
			Type:   "ServiceA",
			Labels: map[string]string{"attr1": "value1"},
		},
		Metrics: []*metricspb.Metric{
			{
				MetricDescriptor: &metricspb.MetricDescriptor{
					Name:        "my-metric",
					Description: "My metric",
					Type:        metricspb.MetricDescriptor_GAUGE_INT64,
				},
				Timeseries: []*metricspb.TimeSeries{
					{
						Points: []*metricspb.Point{
							{Value: &metricspb.Point_Int64Value{Int64Value: 123}},
						},
					},
				},
			},
		},
	}
	assert.NoError(t, lme.ConsumeMetricsData(context.Background(), md))
	assert.NoError(t, lme.Shutdown(context.Background()))

	var j map[string]interface{}
	assert.NoError(t, json.Unmarshal(mf.buf.Bytes(), &j))

	assert.EqualValues(t, j,
		map[string]interface{}{
			"resource": map[string]interface{}{
				"type":   "ServiceA",
				"labels": map[string]interface{}{"attr1": "value1"},
			},
			"metrics": []interface{}{
				map[string]interface{}{
					"metricDescriptor": map[string]interface{}{
						"name":        "my-metric",
						"description": "My metric",
						"type":        "GAUGE_INT64",
					},
					"timeseries": []interface{}{
						map[string]interface{}{
							"points": []interface{}{
								map[string]interface{}{
									"int64Value": "123",
								},
							},
						},
					},
				},
			},
		})
}
