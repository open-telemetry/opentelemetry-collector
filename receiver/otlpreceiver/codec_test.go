// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlpreceiver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/mem"
)

type testUnsafeProtoDecoder struct {
	data []byte
}

func (d *testUnsafeProtoDecoder) UnmarshalProtoUnsafe(data []byte) error {
	d.data = data
	return nil
}

func TestUnsafeGRPCCodecUsesGoOwnedBuffer(t *testing.T) {
	src := []byte("payload")
	dest := &testUnsafeProtoDecoder{}

	require.NoError(t, newUnsafeGRPCCodec().Unmarshal(mem.BufferSlice{mem.SliceBuffer(src)}, dest))
	assert.Equal(t, []byte("payload"), dest.data)

	src[0] = 'P'
	assert.Equal(t, []byte("payload"), dest.data)
}
