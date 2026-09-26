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
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/testdata"
	"go.opentelemetry.io/collector/pdata/xpdata/pref"
)

func TestTracesRequest(t *testing.T) {
	mr := newTracesRequest(testdata.GenerateTraces(1))

	traceErr := consumererror.NewTraces(errors.New("some error"), ptrace.NewTraces())
	assert.Equal(t, newTracesRequest(ptrace.NewTraces()), mr.(request.ErrorHandler).OnError(traceErr))
}

func TestTracesEncodingUnmarshalMarksPipelineOwned(t *testing.T) {
	t.Parallel()

	enc := tracesEncoding{}
	buf, err := enc.Marshal(t.Context(), newTracesRequest(testdata.GenerateTraces(1)))
	require.NoError(t, err)

	_, req, err := enc.Unmarshal(buf)
	require.NoError(t, err)

	td := req.(*tracesRequest).td
	marked := pref.MarkPipelineOwnedTraces(td)
	if marked {
		pref.UnrefTraces(td)
	}
	assert.False(t, marked)
	assert.NotPanics(t, func() { tracesReferenceCounter{}.Unref(req) })
}
