// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pmetricotlp

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gootlpcollectormetrics "go.opentelemetry.io/proto/slim/otlp/collector/metrics/v1"
	goproto "google.golang.org/protobuf/proto"

	"go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/internal/otlp"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

var (
	_ json.Unmarshaler = ExportRequest{}
	_ json.Marshaler   = ExportRequest{}
)

var metricsRequestJSON = []byte(`
	{
		"resourceMetrics": [
			{
				"resource": {},
				"scopeMetrics": [
					{
						"scope": {},
						"metrics": [
							{
								"name": "test_metric"
							}
						]
					}
				]
			}
		]
	}`)

func TestRequestToPData(t *testing.T) {
	tr := NewExportRequest()
	assert.Equal(t, 0, tr.Metrics().MetricCount())
	tr.Metrics().ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
	assert.Equal(t, 1, tr.Metrics().MetricCount())
}

func TestRequestJSON(t *testing.T) {
	mr := NewExportRequest()
	require.NoError(t, mr.UnmarshalJSON(metricsRequestJSON))
	assert.Equal(t, "test_metric", mr.Metrics().ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).Name())

	got, err := mr.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, strings.Join(strings.Fields(string(metricsRequestJSON)), ""), string(got))
}

func TestMetricsProtoWireCompatibility(t *testing.T) {
	// This test verifies that OTLP ProtoBufs generated using goproto lib in
	// opentelemetry-proto repository OTLP ProtoBufs generated using gogoproto lib in
	// this repository are wire compatible.

	// Generate Metrics as pdata struct.
	md := NewExportRequestFromMetrics(pmetric.Metrics(internal.GenTestMetricsWrapper()))

	// Marshal its underlying ProtoBuf to wire.
	wire1, err := md.MarshalProto()
	require.NoError(t, err)
	assert.NotNil(t, wire1)

	// Unmarshal from the wire to OTLP Protobuf in goproto's representation.
	var goprotoMessage gootlpcollectormetrics.ExportMetricsServiceRequest
	err = goproto.Unmarshal(wire1, &goprotoMessage)
	require.NoError(t, err)

	// Marshal to the wire again.
	wire2, err := goproto.Marshal(&goprotoMessage)
	require.NoError(t, err)
	assert.NotNil(t, wire2)

	// Unmarshal from the wire into gogoproto's representation.
	md2 := NewExportRequest()
	err = md2.UnmarshalProto(wire2)
	require.NoError(t, err)

	// Now compare that the original and final ProtoBuf messages are the same.
	// This proves that goproto and gogoproto marshaling/unmarshaling are wire compatible.
	// Migration logic will run, so run it on the original message as well.
	otlp.MigrateMetrics(md.orig.ResourceMetrics)
	assert.Equal(t, md, md2)
}

func TestRejectInvalidUTF8(t *testing.T) {
	t.Run("invalid resource", func(t *testing.T) {
		md := pmetric.NewMetrics()
		rm := md.ResourceMetrics().AppendEmpty()
		rm.Resource().Attributes().PutStr("bad", string([]byte{0xff}))
		rm.ScopeMetrics().AppendEmpty().Metrics().AppendEmpty().SetEmptyGauge().DataPoints().AppendEmpty()

		assert.False(t, NewExportRequestFromMetrics(md).ValidateUTF8())
		assert.Equal(t, 1, NewExportRequestFromMetrics(md).RejectInvalidUTF8())
		assert.Equal(t, 0, md.DataPointCount())
	})

	t.Run("invalid scope", func(t *testing.T) {
		md := pmetric.NewMetrics()
		sm := md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty()
		sm.Scope().SetName(string([]byte{0xff}))
		sm.Metrics().AppendEmpty().SetEmptyGauge().DataPoints().AppendEmpty()

		assert.Equal(t, 1, NewExportRequestFromMetrics(md).RejectInvalidUTF8())
		assert.Equal(t, 0, md.DataPointCount())
	})

	t.Run("invalid metric descriptor", func(t *testing.T) {
		md := pmetric.NewMetrics()
		metric := md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
		metric.SetName(string([]byte{0xff}))
		metric.SetEmptyGauge().DataPoints().AppendEmpty()

		assert.Equal(t, 1, NewExportRequestFromMetrics(md).RejectInvalidUTF8())
		assert.Equal(t, 0, md.DataPointCount())
	})

	for _, tc := range []struct {
		name  string
		build func(pmetric.Metric)
	}{
		{
			name: "gauge datapoint",
			build: func(metric pmetric.Metric) {
				metric.SetEmptyGauge().DataPoints().AppendEmpty().Attributes().PutStr("bad", string([]byte{0xff}))
			},
		},
		{
			name: "sum datapoint",
			build: func(metric pmetric.Metric) {
				metric.SetEmptySum().DataPoints().AppendEmpty().Attributes().PutStr("bad", string([]byte{0xff}))
			},
		},
		{
			name: "histogram datapoint",
			build: func(metric pmetric.Metric) {
				metric.SetEmptyHistogram().DataPoints().AppendEmpty().Attributes().PutStr("bad", string([]byte{0xff}))
			},
		},
		{
			name: "exp histogram datapoint",
			build: func(metric pmetric.Metric) {
				metric.SetEmptyExponentialHistogram().DataPoints().AppendEmpty().Attributes().PutStr("bad", string([]byte{0xff}))
			},
		},
		{
			name: "summary datapoint",
			build: func(metric pmetric.Metric) {
				metric.SetEmptySummary().DataPoints().AppendEmpty().Attributes().PutStr("bad", string([]byte{0xff}))
			},
		},
		{
			name: "unknown metric type",
			build: func(metric pmetric.Metric) {
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			md := pmetric.NewMetrics()
			metric := md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
			tc.build(metric)

			_ = NewExportRequestFromMetrics(md).RejectInvalidUTF8()
		})
	}
}
