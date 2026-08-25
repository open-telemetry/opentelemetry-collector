// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlptext

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestTracesText(t *testing.T) {
	tests := []struct {
		name string
		in   ptrace.Traces
		out  string
	}{
		{
			name: "empty_traces",
			in:   ptrace.NewTraces(),
			out:  "empty.out",
		},
		{
			name: "two_spans",
			in:   testdata.GenerateTraces(2),
			out:  "two_spans.out",
		},
		{
			name: "traces_with_entity_refs",
			in:   generateTracesWithEntityRefs(),
			out:  "traces_with_entity_refs.out",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTextTracesMarshaler().MarshalTraces(tt.in)
			require.NoError(t, err)
			out, err := os.ReadFile(filepath.Join("testdata", "traces", tt.out))
			require.NoError(t, err)
			expected := strings.ReplaceAll(string(out), "\r", "")
			assert.Equal(t, expected, string(got))
		})
	}
}

func generateTracesWithEntityRefs() ptrace.Traces {
	td := ptrace.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()

	setupResourceWithEntityRefs(rs.Resource())

	ss := rs.ScopeSpans().AppendEmpty()
	ss.Scope().SetName("test-scope")
	span := ss.Spans().AppendEmpty()
	span.SetName("test-span")
	span.SetSpanID([8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08})
	span.SetTraceID([16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10})

	return td
}
