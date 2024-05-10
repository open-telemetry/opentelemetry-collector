// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package experr

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewShutdownErr(t *testing.T) {
	err := NewShutdownErr(errors.New("some error"))
	assert.Equal(t, "interrupted due to shutdown: some error", err.Error())
}

func TestIsShutdownErr(t *testing.T) {
	err := errors.New("testError")
	require.False(t, IsShutdownErr(err))
	err = NewShutdownErr(err)
	require.True(t, IsShutdownErr(err))
}
