// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package sendertest

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewNopSenderFunc(t *testing.T) {
	sender := NewNopSenderFunc[int]()
	require.NoError(t, sender(context.Background(), 1))
}

func TestNewErrSenderFunc(t *testing.T) {
	err := errors.New("test")
	sender := NewErrSenderFunc[int](err)
	require.ErrorIs(t, sender(context.Background(), 1), err)
}
