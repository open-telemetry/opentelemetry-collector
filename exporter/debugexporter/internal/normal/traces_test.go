// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package normal

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TestMarshalTraces(t *testing.T) {
	tests := []struct {
		name     string
		input    ptrace.Traces
		expected string
	}{
		{
			name:     "empty traces",
			input:    ptrace.NewTraces(),
			expected: "",
		},
		{
			name: "one span",
			input: func() ptrace.Traces {
				traces := ptrace.NewTraces()
				span := traces.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
				span.SetName("span-name")
				span.SetTraceID([16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10})
				span.SetSpanID([8]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18})
				return traces
			}(),
			expected: `span-name 0102030405060708090a0b0c0d0e0f10 1112131415161718
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := NewNormalTracesMarshaler().MarshalTraces(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, string(output))
		})
	}
}
