// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package iruntime

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadMemInfo(t *testing.T) {
	vmStat, err := readMemInfo()
	assert.NoError(t, err)
	assert.True(t, vmStat > 0)
}
