// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package normal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
			name: "one span with resource and scope attributes",
			input: func() ptrace.Traces {
				traces := ptrace.NewTraces()
				resourceSpans := traces.ResourceSpans().AppendEmpty()
				resourceSpans.SetSchemaUrl("https://opentelemetry.io/resource-schema-url")
				resourceSpans.Resource().Attributes().PutStr("resourceKey1", "resourceValue1")
				resourceSpans.Resource().Attributes().PutBool("resourceKey2", false)
				scopeSpans := resourceSpans.ScopeSpans().AppendEmpty()
				scopeSpans.SetSchemaUrl("http://opentelemetry.io/scope-schema-url")
				scopeSpans.Scope().SetName("scope-name")
				scopeSpans.Scope().SetVersion("1.2.3")
				scopeSpans.Scope().Attributes().PutStr("scopeKey1", "scopeValue1")
				scopeSpans.Scope().Attributes().PutBool("scopeKey2", true)
				span := scopeSpans.Spans().AppendEmpty()
				span.SetName("span-name")
				span.SetTraceID([16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10})
				span.SetSpanID([8]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18})
				span.Attributes().PutStr("key1", "value1")
				span.Attributes().PutStr("key2", "value2")
				return traces
			}(),
			expected: `ResourceTraces #0 [https://opentelemetry.io/resource-schema-url] resourceKey1=resourceValue1 resourceKey2=false
ScopeTraces #0 scope-name@1.2.3 [http://opentelemetry.io/scope-schema-url] scopeKey1=scopeValue1 scopeKey2=true
span-name 0102030405060708090a0b0c0d0e0f10 1112131415161718 key1=value1 key2=value2
`,
		},
		{
			name: "one span",
			input: func() ptrace.Traces {
				traces := ptrace.NewTraces()
				span := traces.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
				span.SetName("span-name")
				span.SetTraceID([16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10})
				span.SetSpanID([8]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18})
				span.Attributes().PutStr("key1", "value1")
				span.Attributes().PutStr("key2", "value2")
				return traces
			}(),
			expected: `ResourceTraces #0
ScopeTraces #0
span-name 0102030405060708090a0b0c0d0e0f10 1112131415161718 key1=value1 key2=value2
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := NewNormalTracesMarshaler().MarshalTraces(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(output))
		})
	}
}
