// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scrapererror

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPartialScrapeError(t *testing.T) {
	failed := 2
	err := errors.New("some error")
	partialErr := NewPartialScrapeError(err, failed)
	require.EqualError(t, err, partialErr.Error())
	assert.Equal(t, failed, partialErr.Failed)
}

func TestIsPartialScrapeError(t *testing.T) {
	err := errors.New("testError")
	require.False(t, IsPartialScrapeError(err))

	err = NewPartialScrapeError(err, 2)
	require.True(t, IsPartialScrapeError(err))
}
