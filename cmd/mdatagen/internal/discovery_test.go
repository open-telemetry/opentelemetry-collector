// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscoveryValidate(t *testing.T) {
	valid := map[string]DiscoveryEndpoint{"port": {Rule: "port == 6379"}}
	tests := []struct {
		name      string
		class     string
		discovery Discovery
		wantErr   string
	}{
		{name: "receiver", class: "receiver", discovery: Discovery{Endpoints: valid}},
		{name: "scraper", class: "scraper", discovery: Discovery{Endpoints: valid}},
		{name: "unsupported class", class: "exporter", discovery: Discovery{Endpoints: valid}, wantErr: `discovery is only supported for receivers and scrapers, got class "exporter"`},
		{name: "no endpoints", class: "scraper", discovery: Discovery{}, wantErr: "discovery.endpoints must have at least one endpoint type"},
		{name: "empty rule", class: "scraper", discovery: Discovery{Endpoints: map[string]DiscoveryEndpoint{"port": {Rule: " "}}}, wantErr: "discovery.endpoints.port.rule must be set"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.discovery.validate(tt.class)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestDiscoverySortedEndpointTypes(t *testing.T) {
	d := Discovery{Endpoints: map[string]DiscoveryEndpoint{"port": {}, "container": {}, "hostport": {}}}
	assert.Equal(t, []string{"container", "hostport", "port"}, d.SortedEndpointTypes())
}

func TestDiscoveryDefaultConfigLiteral(t *testing.T) {
	tests := []struct {
		name   string
		config map[string]any
		want   string
	}{
		{name: "empty", config: nil, want: "map[string]any{}"},
		{name: "flat", config: map[string]any{"b": 1, "a": "`endpoint`"}, want: "map[string]any{\"a\":\"`endpoint`\", \"b\":1}"},
		{
			name:   "nested",
			config: map[string]any{"hosts": []any{map[string]any{"endpoint": "`endpoint`"}}},
			want:   "map[string]any{\"hosts\":[]any{map[string]any{\"endpoint\":\"`endpoint`\"}}}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := Discovery{DefaultConfig: tt.config}
			assert.Equal(t, tt.want, d.DefaultConfigLiteral())
		})
	}
}
