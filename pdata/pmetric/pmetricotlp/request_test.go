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

func TestSanitizeUTF8(t *testing.T) {
	md := pmetric.NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()
	rm.Resource().Attributes().PutStr("bad\xff", "bad\xff")
	sm := rm.ScopeMetrics().AppendEmpty()
	sm.Scope().SetName("scope\xff")
	m := sm.Metrics().AppendEmpty()
	m.SetName("metric\xff")
	m.SetUnit("unit\xff")
	dp := m.SetEmptyGauge().DataPoints().AppendEmpty()
	dp.Attributes().PutStr("dp\xff", "dp\xff")
	dp.Exemplars().AppendEmpty().FilteredAttributes().PutStr("ex\xff", "ex\xff")
	m = sm.Metrics().AppendEmpty()
	m.SetName("summary")
	m.SetEmptySummary().DataPoints().AppendEmpty().Attributes().PutStr("s\xff", "s\xff")

	NewExportRequestFromMetrics(md).SanitizeUTF8()

	expected := pmetric.NewMetrics()
	erm := expected.ResourceMetrics().AppendEmpty()
	erm.Resource().Attributes().PutStr("bad�", "bad�")
	esm := erm.ScopeMetrics().AppendEmpty()
	esm.Scope().SetName("scope�")
	em := esm.Metrics().AppendEmpty()
	em.SetName("metric�")
	em.SetUnit("unit�")
	edp := em.SetEmptyGauge().DataPoints().AppendEmpty()
	edp.Attributes().PutStr("dp�", "dp�")
	edp.Exemplars().AppendEmpty().FilteredAttributes().PutStr("ex�", "ex�")
	em = esm.Metrics().AppendEmpty()
	em.SetName("summary")
	em.SetEmptySummary().DataPoints().AppendEmpty().Attributes().PutStr("s�", "s�")
	assert.Equal(t, expected, md)
}

func TestSanitizeUTF8ReadOnly(t *testing.T) {
	md := pmetric.NewMetrics()
	md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty().SetName("bad\xff")
	md.MarkReadOnly()
	assert.Panics(t, func() { NewExportRequestFromMetrics(md).SanitizeUTF8() })
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

	t.Run("invalid resource schema url", func(t *testing.T) {
		md := pmetric.NewMetrics()
		rm := md.ResourceMetrics().AppendEmpty()
		rm.SetSchemaUrl(string([]byte{0xff}))
		rm.ScopeMetrics().AppendEmpty().Metrics().AppendEmpty().SetEmptyGauge().DataPoints().AppendEmpty()

		assert.False(t, NewExportRequestFromMetrics(md).ValidateUTF8())
		assert.Equal(t, 1, NewExportRequestFromMetrics(md).RejectInvalidUTF8())
		assert.Equal(t, 0, md.DataPointCount())
	})

	t.Run("invalid scope schema url", func(t *testing.T) {
		md := pmetric.NewMetrics()
		sm := md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty()
		sm.SetSchemaUrl(string([]byte{0xff}))
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

	t.Run("invalid metric metadata", func(t *testing.T) {
		md := pmetric.NewMetrics()
		metric := md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
		metric.SetName("valid")
		metric.Metadata().PutStr("bad", string([]byte{0xff}))
		metric.SetEmptyGauge().DataPoints().AppendEmpty()
		metric.Gauge().DataPoints().AppendEmpty()

		assert.False(t, NewExportRequestFromMetrics(md).ValidateUTF8())
		assert.Equal(t, 2, NewExportRequestFromMetrics(md).RejectInvalidUTF8())
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
	} {
		t.Run(tc.name, func(t *testing.T) {
			md := pmetric.NewMetrics()
			sm := md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty()
			tc.build(sm.Metrics().AppendEmpty())

			assert.Equal(t, 1, NewExportRequestFromMetrics(md).RejectInvalidUTF8())
			assert.Equal(t, 0, md.DataPointCount())
			// The metric lost its only data point, so it is dropped together with its empty parents.
			assert.Equal(t, 0, md.ResourceMetrics().Len())
		})
	}

	t.Run("valid metric without data points is kept", func(t *testing.T) {
		md := pmetric.NewMetrics()
		sm := md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty()
		sm.Metrics().AppendEmpty().SetName("no-data-points")
		sm.Metrics().AppendEmpty().SetEmptyGauge()

		assert.Equal(t, 0, NewExportRequestFromMetrics(md).RejectInvalidUTF8())
		assert.Equal(t, 2, sm.Metrics().Len())
	})

	t.Run("empty containers are kept", func(t *testing.T) {
		md := pmetric.NewMetrics()
		md.ResourceMetrics().AppendEmpty()
		md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty()

		assert.Equal(t, 0, NewExportRequestFromMetrics(md).RejectInvalidUTF8())
		assert.Equal(t, 2, md.ResourceMetrics().Len())
		assert.Equal(t, 1, md.ResourceMetrics().At(1).ScopeMetrics().Len())
	})
}

func BenchmarkRejectInvalidUTF8(b *testing.B) {
	const (
		metrics         = 100
		pointsPerMetric = 100
		dataPoints      = metrics * pointsPerMetric
	)
	newRequest := func(invalidEvery int) ExportRequest {
		req := NewExportRequest()
		rm := req.Metrics().ResourceMetrics().AppendEmpty()
		rm.Resource().Attributes().PutStr("service.name", "svc")
		sm := rm.ScopeMetrics().AppendEmpty()
		sm.Scope().SetName("scope")
		ms := sm.Metrics()
		ms.EnsureCapacity(metrics)
		for m := range metrics {
			metric := ms.AppendEmpty()
			metric.SetName("http.server.request.duration")
			metric.SetDescription("Duration of HTTP server requests")
			metric.SetUnit("s")
			dps := metric.SetEmptyGauge().DataPoints()
			dps.EnsureCapacity(pointsPerMetric)
			for p := range pointsPerMetric {
				dp := dps.AppendEmpty()
				dp.SetDoubleValue(1)
				dp.Attributes().PutStr("http.method", "GET")
				// The invalid string is the last one of the data point, so the whole data point is checked.
				route := "/api/v1/resource"
				if i := m*pointsPerMetric + p; invalidEvery > 0 && i%invalidEvery == 0 {
					route = "/api/\xff/resource"
				}
				dp.Attributes().PutStr("http.route", route)
			}
		}
		return req
	}

	for _, bc := range []struct {
		name         string
		invalidEvery int
		want         int
	}{
		{name: "AllValid", invalidEvery: 0, want: 0},
		{name: "OneInvalid", invalidEvery: dataPoints, want: 1},
		{name: "AllInvalid", invalidEvery: 1, want: dataPoints},
	} {
		b.Run(bc.name, func(b *testing.B) {
			req := newRequest(bc.invalidEvery)
			b.ReportAllocs()
			for b.Loop() {
				if got := req.RejectInvalidUTF8(); got != bc.want {
					b.Fatalf("rejected %d, want %d", got, bc.want)
				}
				if bc.invalidEvery > 0 {
					// Rejection removed the invalid items, so build the request again.
					b.StopTimer()
					req = newRequest(bc.invalidEvery)
					b.StartTimer()
				}
			}
		})
	}
}
