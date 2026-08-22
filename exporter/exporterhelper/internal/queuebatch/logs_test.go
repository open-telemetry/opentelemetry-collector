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
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/testdata"
	"go.opentelemetry.io/collector/pdata/xpdata/pref"
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

func TestLogsEncodingUnmarshalMarksPipelineOwned(t *testing.T) {
	t.Parallel()

	enc := logsEncoding{}
	buf, err := enc.Marshal(t.Context(), newLogsRequest(testdata.GenerateLogs(1)))
	require.NoError(t, err)

	_, req, err := enc.Unmarshal(buf)
	require.NoError(t, err)

	ld := req.(*logsRequest).ld
	marked := pref.MarkPipelineOwnedLogs(ld)
	if marked {
		pref.UnrefLogs(ld)
	}
	assert.False(t, marked)
	assert.NotPanics(t, func() { logsReferenceCounter{}.Unref(req) })
}
