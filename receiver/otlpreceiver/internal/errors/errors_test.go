// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package errors // import "go.opentelemetry.io/collector/receiver/otlpreceiver/internal/util"

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/collector/consumer/consumererror"
)

func Test_GetStatusFromError(t *testing.T) {
	tests := []struct {
		name     string
		input    error
		expected *status.Status
	}{
		{
			name:     "Status",
			input:    status.Error(codes.Aborted, "test"),
			expected: status.New(codes.Aborted, "test"),
		},
		{
			name:     "Permanent Error",
			input:    consumererror.NewPermanent(fmt.Errorf("test")),
			expected: status.New(codes.Internal, "Permanent error: test"),
		},
		{
			name:     "Non-Permanent Error",
			input:    fmt.Errorf("test"),
			expected: status.New(codes.Unavailable, "test"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetStatusFromError(tt.input)
			assert.Equal(t, tt.expected.Err(), result)
		})
	}
}
