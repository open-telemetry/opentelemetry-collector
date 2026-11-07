// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsconsumer

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "context canceled",
			err:      context.Canceled,
			expected: componentErrorTypeCanceled,
		},
		{
			name:     "wrapped context canceled",
			err:      fmt.Errorf("consume failed: %w", context.Canceled),
			expected: componentErrorTypeCanceled,
		},
		{
			name:     "context deadline exceeded",
			err:      context.DeadlineExceeded,
			expected: componentErrorTypeDeadlineExceeded,
		},
		{
			name:     "wrapped context deadline exceeded",
			err:      fmt.Errorf("consume failed: %w", context.DeadlineExceeded),
			expected: componentErrorTypeDeadlineExceeded,
		},
		{
			name:     "ordinary error",
			err:      errors.New("consumer failed"),
			expected: componentErrorTypeUnspecified,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: componentErrorTypeUnspecified,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, classifyError(tt.err))
		})
	}
}
