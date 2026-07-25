// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/confmap"
)

func TestRedactWithPreExpansion(t *testing.T) {
	tests := []struct {
		name     string
		pre      map[string]any
		redacted map[string]any
		want     map[string]any
	}{
		{
			name: "redact hardcoded opaque value",
			pre: map[string]any{
				"exporters": map[string]any{
					"foo": map[string]any{
						"token": "secret",
						"other": "hello",
					},
				},
			},
			redacted: map[string]any{
				"exporters": map[string]any{
					"foo": map[string]any{
						"token": redactedMask,
						"other": "hello",
					},
				},
			},
			want: map[string]any{
				"exporters": map[string]any{
					"foo": map[string]any{
						"token": redactedMask,
						"other": "hello",
					},
				},
			},
		},
		{
			name: "preserve provider reference at opaque leaf",
			pre: map[string]any{
				"exporters": map[string]any{
					"foo": map[string]any{
						"token": "${env:TOKEN}",
					},
				},
			},
			redacted: map[string]any{
				"exporters": map[string]any{
					"foo": map[string]any{
						"token": redactedMask,
					},
				},
			},
			want: map[string]any{
				"exporters": map[string]any{
					"foo": map[string]any{
						"token": "${env:TOKEN}",
					},
				},
			},
		},
		{
			name: "redact embedded provider reference at opaque leaf",
			pre: map[string]any{
				"exporters": map[string]any{
					"foo": map[string]any{
						"token": "Bearer ${env:TOKEN}",
					},
				},
			},
			redacted: map[string]any{
				"exporters": map[string]any{
					"foo": map[string]any{
						"token": redactedMask,
					},
				},
			},
			want: map[string]any{
				"exporters": map[string]any{
					"foo": map[string]any{
						"token": redactedMask,
					},
				},
			},
		},
		{
			name: "redact provider reference with hardcoded suffix at opaque leaf",
			pre: map[string]any{
				"exporters": map[string]any{
					"foo": map[string]any{
						"token": "${env:TOKEN} hardcoded-secret",
					},
				},
			},
			redacted: map[string]any{
				"exporters": map[string]any{
					"foo": map[string]any{
						"token": redactedMask,
					},
				},
			},
			want: map[string]any{
				"exporters": map[string]any{
					"foo": map[string]any{
						"token": redactedMask,
					},
				},
			},
		},
		{
			name: "redact nested map and slice",
			pre: map[string]any{
				"exporters": map[string]any{
					"foo": map[string]any{
						"nested": map[string]any{
							"secret": "abc",
							"list":   []any{"a", "${env:B}", "c"},
						},
					},
				},
			},
			redacted: map[string]any{
				"exporters": map[string]any{
					"foo": map[string]any{
						"nested": map[string]any{
							"secret": redactedMask,
							"list":   []any{redactedMask, redactedMask, redactedMask},
						},
					},
				},
			},
			want: map[string]any{
				"exporters": map[string]any{
					"foo": map[string]any{
						"nested": map[string]any{
							"secret": redactedMask,
							"list":   []any{redactedMask, "${env:B}", redactedMask},
						},
					},
				},
			},
		},
		{
			name: "non-opaque strings untouched",
			pre: map[string]any{
				"exporters": map[string]any{
					"foo": map[string]any{
						// Value happens to equal the masked string but is not an
						// opaque field — the redacted view confirms that by
						// carrying the same plain string back.
						"other": redactedMask,
					},
				},
			},
			redacted: map[string]any{
				"exporters": map[string]any{
					"foo": map[string]any{
						"other": redactedMask,
					},
				},
			},
			want: map[string]any{
				"exporters": map[string]any{
					"foo": map[string]any{
						"other": redactedMask,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pre := confmap.NewFromStringMap(tt.pre)
			red := confmap.NewFromStringMap(tt.redacted)
			got := redactByMirroring(pre, red)
			require.NotNil(t, got)
			assert.Equal(t, tt.want, got.ToStringMap())
		})
	}
}

func TestRedactWithPreExpansion_NilPreExpansion(t *testing.T) {
	got := redactByMirroring(nil, confmap.New())
	assert.Nil(t, got)
}

func TestRedactWithPreExpansion_NilRedacted(t *testing.T) {
	pre := confmap.NewFromStringMap(map[string]any{"a": "b"})
	got := redactByMirroring(pre, nil)
	require.NotNil(t, got)
	assert.Equal(t, map[string]any{"a": "b"}, got.ToStringMap())
}
