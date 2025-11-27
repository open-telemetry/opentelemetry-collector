// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestLogsRequest(t *testing.T) {
	lr := newLogsRequest(testdata.GenerateLogs(1))

	logErr := consumererror.NewLogs(errors.New("some error"), plog.NewLogs())
	assert.Equal(
		t,
		newLogsRequest(plog.NewLogs()),
		lr.(request.ErrorHandler).OnError(logErr),
	)
}
