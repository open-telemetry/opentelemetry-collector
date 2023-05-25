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

	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/pdata/ptrace"
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTextTracesMarshaler().MarshalTraces(tt.in)
			assert.NoError(t, err)
			out, err := os.ReadFile(filepath.Join("testdata", "traces", tt.out))
			require.NoError(t, err)
			expected := strings.ReplaceAll(string(out), "\r", "")
			assert.Equal(t, expected, string(got))
		})
	}
}
