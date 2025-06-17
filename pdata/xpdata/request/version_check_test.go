// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package request

import (
	"context"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TestIsRequestPayloadV1(t *testing.T) {
	// too short
	assert.False(t, isRequestPayloadV1([]byte{protoTag1TypeByte, 0x00}))

	buf := make([]byte, 5)

	// wrong type: field 1, wire type 2 (length-delimited)
	buf[0] = 0x0A
	assert.False(t, isRequestPayloadV1([]byte{protoTag1TypeByte, 0x00}))

	// wrong version
	buf[0] = protoTag1TypeByte
	binary.LittleEndian.PutUint32(buf[1:], 2)
	assert.False(t, isRequestPayloadV1(buf))

	binary.LittleEndian.PutUint32(buf[1:], requestFormatVersion)
	assert.True(t, isRequestPayloadV1(buf))

	buf, err := MarshalMetrics(context.Background(), pmetric.NewMetrics())
	require.NoError(t, err)
	assert.True(t, isRequestPayloadV1(buf))

	buf, err = MarshalTraces(context.Background(), ptrace.NewTraces())
	require.NoError(t, err)
	assert.True(t, isRequestPayloadV1(buf))

	buf, err = MarshalLogs(context.Background(), plog.NewLogs())
	require.NoError(t, err)
	assert.True(t, isRequestPayloadV1(buf))

	buf, err = MarshalProfiles(context.Background(), pprofile.NewProfiles())
	require.NoError(t, err)
	assert.True(t, isRequestPayloadV1(buf))
}
