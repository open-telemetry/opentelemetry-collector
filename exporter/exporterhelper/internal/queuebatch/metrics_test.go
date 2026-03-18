// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestMetricsRequest(t *testing.T) {
	mr := newMetricsRequest(testdata.GenerateMetrics(1))
	req := mr.(*metricsRequest)
	req.cachedSize = 123
	
	remaining := pmetric.NewMetrics()
	metricsErr := consumererror.NewMetrics(errors.New("some error"), remaining)
	handled := mr.(request.ErrorHandler).OnError(metricsErr)

	assert.Same(t, req, handled)
	assert.Equal(t, remaining, req.md)
	assert.Equal(t, -1, req.cachedSize)
}
