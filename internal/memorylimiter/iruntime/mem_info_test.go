// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package iruntime

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadMemInfo(t *testing.T) {
	vmStat, err := readMemInfo()
	require.NoError(t, err)
	assert.Positive(t, vmStat)
}
