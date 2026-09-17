// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSendingQueueValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  SendingQueue
		wantErr string
	}{
		{
			name:   "default",
			config: SendingQueue{Support: SendingQueueSupportDefault},
		},
		{
			name: "default with overrides",
			config: SendingQueue{
				Support:   SendingQueueSupportDefault,
				Overrides: map[string]any{"num_consumers": 1},
			},
			wantErr: "sending_queue.overrides requires support to be has_overrides",
		},
		{
			name: "has overrides",
			config: SendingQueue{
				Support: SendingQueueSupportHasOverrides,
				Overrides: map[string]any{
					"num_consumers":     1,
					"wait_for_result":   true,
					"block_on_overflow": true,
				},
			},
		},
		{
			name:    "has overrides without overrides",
			config:  SendingQueue{Support: SendingQueueSupportHasOverrides},
			wantErr: "sending_queue.overrides is required when support is has_overrides",
		},
		{
			name: "disabled with rationale",
			config: SendingQueue{
				Support:   SendingQueueSupportHasOverrides,
				Rationale: "Preserve existing behavior.",
				Overrides: map[string]any{"enabled": false},
			},
		},
		{
			name: "disabled without rationale",
			config: SendingQueue{
				Support:   SendingQueueSupportHasOverrides,
				Overrides: map[string]any{"enabled": false},
			},
			wantErr: "sending_queue.rationale is required when overrides disable the queue",
		},
		{
			name: "omitted",
			config: SendingQueue{
				Support:   SendingQueueSupportOmitted,
				Rationale: "No sender.",
			},
		},
		{
			name:    "omitted without rationale",
			config:  SendingQueue{Support: SendingQueueSupportOmitted},
			wantErr: "sending_queue.rationale is required when support is omitted",
		},
		{
			name: "omitted with overrides",
			config: SendingQueue{
				Support:   SendingQueueSupportOmitted,
				Rationale: "No sender.",
				Overrides: map[string]any{"num_consumers": 1},
			},
			wantErr: "sending_queue.overrides cannot be set when support is omitted",
		},
		{
			name:    "invalid support",
			config:  SendingQueue{Support: "sometimes"},
			wantErr: "sending_queue.support must be one of",
		},
		{
			name: "unknown override",
			config: SendingQueue{
				Support:   SendingQueueSupportHasOverrides,
				Overrides: map[string]any{"unknown": true},
			},
			wantErr: "invalid sending_queue.overrides",
		},
		{
			name: "invalid enabled override",
			config: SendingQueue{
				Support:   SendingQueueSupportHasOverrides,
				Overrides: map[string]any{"enabled": "false"},
			},
			wantErr: "enabled must be a boolean",
		},
		{
			name: "invalid override value",
			config: SendingQueue{
				Support:   SendingQueueSupportHasOverrides,
				Overrides: map[string]any{"num_consumers": 0},
			},
			wantErr: "`num_consumers` must be positive",
		},
		{
			name: "invalid nested batch override",
			config: SendingQueue{
				Support: SendingQueueSupportHasOverrides,
				Overrides: map[string]any{
					"batch": map[string]any{
						"min_size": 12,
						"max_size": 10,
					},
				},
			},
			wantErr: "`max_size` (10) must be greater or equal to `min_size` (12)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

func TestSendingQueueGoQueueOverrides(t *testing.T) {
	config := SendingQueue{
		Overrides: map[string]any{
			"enabled":       false,
			"num_consumers": 1,
			"batch": map[string]any{
				"enabled":  false,
				"min_size": int64(12),
			},
		},
	}

	actual, err := config.GoQueueOverrides()
	require.NoError(t, err)
	require.Equal(t, `map[string]any{"batch": map[string]any{"enabled": false, "min_size": 12}, "num_consumers": 1}`, actual)

	actualYAML, err := config.IndentedYAMLQueueOverrides()
	require.NoError(t, err)
	require.NotContains(t, actualYAML, "\n  enabled: false")
	require.Contains(t, actualYAML, "num_consumers: 1")
}
