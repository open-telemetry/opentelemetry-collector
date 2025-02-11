// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExportSenderRightArguments(t *testing.T) {
	es := newSender[int64](func(_ context.Context, data int64) error {
		assert.Equal(t, int64(1), data)
		return nil
	})
	require.NoError(t, es.Send(context.Background(), int64(1)))
}

func TestExportSenderReturnsError(t *testing.T) {
	err := errors.New("test error")
	es := newSender[int64](func(_ context.Context, data int64) error {
		assert.Equal(t, int64(1), data)
		return err
	})
	require.ErrorIs(t, es.Send(context.Background(), int64(1)), err)
}
