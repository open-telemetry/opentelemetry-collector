// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pmetricotlp

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
