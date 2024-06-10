// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_zstdCodec_CompressDecompress(t *testing.T) {
	// prepare
	msg := []byte("Hello world.")
	compressed := &bytes.Buffer{}

	// zstd header, for sanity checking
	header := []byte{40, 181, 47, 253}

	c := NewZstdCodec()
	cWriter, err := c.Compress(compressed)
	require.NoError(t, err)
	require.NotNil(t, cWriter)

	_, err = cWriter.Write(msg)
	require.NoError(t, err)
	cWriter.Close()

	cReader, err := c.Decompress(compressed)
	require.NoError(t, err)
	require.NotNil(t, cReader)

	uncompressed, err := io.ReadAll(cReader)
	require.NoError(t, err)
	require.Equal(t, msg, uncompressed)

	// test header
	require.Equal(t, header, compressed.Bytes()[:4])
}
