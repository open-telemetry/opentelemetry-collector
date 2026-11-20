// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !linux

package reclaim

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestColdBinaryPagesUnsupportedOS(t *testing.T) {
	t.Parallel()

	res, err := ColdBinaryPages()
	require.NoError(t, err)
	assert.Equal(t, "unsupported operating system", res.SkippedReason)
	assert.Zero(t, res.RangesAdvised)
	assert.Zero(t, res.BytesAdvised)
}
