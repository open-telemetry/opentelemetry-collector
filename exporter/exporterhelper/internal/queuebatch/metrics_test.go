// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/testdata"
	"go.opentelemetry.io/collector/pdata/xpdata/pref"
)

func TestMetricsRequest(t *testing.T) {
	mr := newMetricsRequest(testdata.GenerateMetrics(1))

	metricsErr := consumererror.NewMetrics(errors.New("some error"), pmetric.NewMetrics())
	assert.Equal(
		t,
		newMetricsRequest(pmetric.NewMetrics()),
		mr.(request.ErrorHandler).OnError(metricsErr),
	)
}

func TestMetricsEncodingUnmarshalMarksPipelineOwned(t *testing.T) {
	t.Parallel()

	enc := metricsEncoding{}
	buf, err := enc.Marshal(t.Context(), newMetricsRequest(testdata.GenerateMetrics(1)))
	require.NoError(t, err)

	_, req, err := enc.Unmarshal(buf)
	require.NoError(t, err)

	md := req.(*metricsRequest).md
	marked := pref.MarkPipelineOwnedMetrics(md)
	if marked {
		pref.UnrefMetrics(md)
	}
	assert.False(t, marked)
	assert.NotPanics(t, func() { metricsReferenceCounter{}.Unref(req) })
}
