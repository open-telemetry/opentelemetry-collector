// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package fromconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseComponents(t *testing.T) {
	for _, tt := range []struct {
		name    string
		config  string
		want    []Component
		wantErr string
	}{
		{
			name: "all classes",
			config: `
receivers:
  otlp:
processors:
  batch:
exporters:
  debug:
extensions:
  file_storage:
connectors:
  forward:
service:
  pipelines: {}
`,
			want: []Component{
				{Class: Receiver, Type: "otlp"},
				{Class: Processor, Type: "batch"},
				{Class: Exporter, Type: "debug"},
				{Class: Extension, Type: "file_storage"},
				{Class: Connector, Type: "forward"},
			},
		},
		{
			name: "named IDs are deduplicated by type",
			config: `
receivers:
  otlp:
  otlp/2:
  otlp/with/slash:
`,
			want: []Component{{Class: Receiver, Type: "otlp"}},
		},
		{
			name: "sorted by type within class",
			config: `
receivers:
  zipkin:
  hostmetrics:
`,
			want: []Component{{Class: Receiver, Type: "hostmetrics"}, {Class: Receiver, Type: "zipkin"}},
		},
		{
			name: "null and absent sections",
			config: `
receivers:
exporters: null
`,
			want: nil,
		},
		{
			name:   "empty document",
			config: ``,
			want:   nil,
		},
		{
			name: "environment variables in keys are not expanded",
			config: `
receivers:
  ${env:FOO}:
`,
			want: []Component{{Class: Receiver, Type: "${env:FOO}"}},
		},
		{
			name: "section from an environment variable",
			config: `
receivers: ${env:RECEIVERS}
`,
			wantErr: `section "receivers" must be a mapping`,
		},
		{
			name:    "invalid yaml",
			config:  "receivers: [",
			wantErr: "failed to parse configuration",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseComponents([]byte(tt.config))
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
