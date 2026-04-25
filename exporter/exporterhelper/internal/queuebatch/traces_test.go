// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestTracesRequestOnError(t *testing.T) {
	tr := newTracesRequest(testdata.GenerateTraces(1))
	req := tr.(*tracesRequest)
	req.cachedSize = 123

	remaining := ptrace.NewTraces()
	traceErr := consumererror.NewTraces(errors.New("some error"), remaining)
	handled := tr.(request.ErrorHandler).OnError(traceErr)

	assert.Same(t, req, handled)
	assert.Equal(t, remaining, req.td)
	assert.Equal(t, -1, req.cachedSize)
}
