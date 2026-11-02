// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package supportbundle

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPProfGatherer(t *testing.T) {
	files, err := PProfGatherer{}.Gather(context.Background())
	require.NoError(t, err)

	require.Len(t, files, 2)
	assert.Equal(t, "pprof/heap.pprof", files[0].Path)
	assert.Equal(t, "pprof/goroutine.pprof", files[1].Path)
	for _, f := range files {
		// The pprof format is a gzipped protobuf.
		zr, err := gzip.NewReader(bytes.NewReader(f.Content))
		require.NoError(t, err, f.Path)
		raw, err := io.ReadAll(zr)
		require.NoError(t, err, f.Path)
		assert.NotEmpty(t, raw, f.Path)
	}
}

func TestPProfGathererCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := PProfGatherer{}.Gather(ctx)
	assert.ErrorIs(t, err, context.Canceled)
}
