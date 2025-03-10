// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package limiter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNopLimiter(t *testing.T) {
	ctx := context.Background()
	nop := NewNop()
	rel, err := nop.Acquire(ctx, 1000)
	require.NoError(t, err)
	rel()
}
