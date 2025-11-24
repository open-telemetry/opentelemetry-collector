// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componenttest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNopHost(t *testing.T) {
	nh := NewNopHost()
	require.NotNil(t, nh)
	require.IsType(t, &nopHost{}, nh)

	extensions := nh.GetExtensions()
	assert.NotNil(t, extensions)
	assert.Empty(t, extensions)
}
