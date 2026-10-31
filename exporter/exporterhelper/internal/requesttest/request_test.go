// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package requesttest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
)

func TestFakeRequestMergeSplitRequests(t *testing.T) {
	first := &FakeRequest{Items: 2, Bytes: 20}
	second := &FakeRequest{Items: 3, Bytes: 30}

	merged, err := first.MergeSplit(t.Context(), 1, request.SizerTypeRequests, second)

	require.NoError(t, err)
	require.Len(t, merged, 1)
	assert.Same(t, first, merged[0])
	assert.Equal(t, 5, first.Items)
	assert.Equal(t, 50, first.Bytes)
}
