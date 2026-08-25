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

func TestTracesRequest(t *testing.T) {
	mr := newTracesRequest(testdata.GenerateTraces(1))

	traceErr := consumererror.NewTraces(errors.New("some error"), ptrace.NewTraces())
	assert.Equal(t, newTracesRequest(ptrace.NewTraces()), mr.(request.ErrorHandler).OnError(traceErr))
}
