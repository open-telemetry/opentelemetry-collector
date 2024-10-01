// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package processorhelper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildProcessorCustomMetricName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "firstMeasure",
			want: "processor_test_type_firstMeasure",
		},
		{
			name: "secondMeasure",
			want: "processor_test_type_secondMeasure",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildCustomMetricName("test_type", tt.name)
			assert.Equal(t, tt.want, got)
		})
	}
}
