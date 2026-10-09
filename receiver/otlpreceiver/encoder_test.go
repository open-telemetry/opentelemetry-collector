// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlpreceiver

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
)

func TestProtoEncoderUnsafeUnmarshal(t *testing.T) {
	const body = "test-body"
	src := plogotlp.NewExportRequest()
	src.Logs().ResourceLogs().AppendEmpty().
		ScopeLogs().AppendEmpty().
		LogRecords().AppendEmpty().
		Body().SetStr(body)
	buf, err := src.MarshalProto()
	require.NoError(t, err)

	dest, err := pbUnsafeEncoder.unmarshalLogsRequest(buf)
	require.NoError(t, err)
	gotBody := dest.Logs().ResourceLogs().At(0).
		ScopeLogs().At(0).
		LogRecords().At(0).
		Body()
	assert.Equal(t, body, gotBody.Str())

	pos := bytes.Index(buf, []byte(body))
	require.NotEqual(t, -1, pos)
	buf[pos] = 'T'
	assert.Equal(t, "Test-body", gotBody.Str())
}

func TestProtoEncoderSafeUnmarshalCopies(t *testing.T) {
	const body = "test-body"
	src := plogotlp.NewExportRequest()
	src.Logs().ResourceLogs().AppendEmpty().
		ScopeLogs().AppendEmpty().
		LogRecords().AppendEmpty().
		Body().SetStr(body)
	buf, err := src.MarshalProto()
	require.NoError(t, err)

	dest, err := pbEncoder.unmarshalLogsRequest(buf)
	require.NoError(t, err)
	gotBody := dest.Logs().ResourceLogs().At(0).
		ScopeLogs().At(0).
		LogRecords().At(0).
		Body()

	pos := bytes.Index(buf, []byte(body))
	require.NotEqual(t, -1, pos)
	buf[pos] = 'T'
	assert.Equal(t, body, gotBody.Str())
}
