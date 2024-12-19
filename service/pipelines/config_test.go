// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pipelines

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
)

func TestConfigValidate(t *testing.T) {
	testCases := []struct {
		name     string // test case name (also file name containing config yaml)
		cfgFn    func(*testing.T) Config
		expected error
	}{
		{
			name:     "valid",
			cfgFn:    generateConfig,
			expected: nil,
		},
		{
			name: "duplicate-processor-reference",
			cfgFn: func(*testing.T) Config {
				cfg := generateConfig(t)
				pipe := cfg[pipeline.NewID(pipeline.SignalTraces)]
				pipe.Processors = append(pipe.Processors, pipe.Processors...)
				return cfg
			},
			expected: fmt.Errorf(`pipeline "traces": %w`, errors.New(`references processor "nop" multiple times`)),
		},
		{
			name: "missing-pipeline-receivers",
			cfgFn: func(*testing.T) Config {
				cfg := generateConfig(t)
				cfg[pipeline.NewID(pipeline.SignalTraces)].Receivers = nil
				return cfg
			},
			expected: fmt.Errorf(`pipeline "traces": %w`, errMissingServicePipelineReceivers),
		},
		{
			name: "missing-pipeline-exporters",
			cfgFn: func(*testing.T) Config {
				cfg := generateConfig(t)
				cfg[pipeline.NewID(pipeline.SignalTraces)].Exporters = nil
				return cfg
			},
			expected: fmt.Errorf(`pipeline "traces": %w`, errMissingServicePipelineExporters),
		},
		{
			name: "missing-pipelines",
			cfgFn: func(*testing.T) Config {
				return nil
			},
			expected: errMissingServicePipelines,
		},
		{
			name: "invalid-service-pipeline-type",
			cfgFn: func(*testing.T) Config {
				cfg := generateConfig(t)
				cfg[pipeline.MustNewID("wrongtype")] = &PipelineConfig{
					Receivers:  []component.ID{component.MustNewID("nop")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewID("nop")},
				}
				return cfg
			},
			expected: errors.New(`pipeline "wrongtype": unknown signal "wrongtype"`),
		},
		{
			name: "disabled-featuregate-profiles",
			cfgFn: func(*testing.T) Config {
				cfg := generateConfig(t)
				cfg[pipeline.NewID(xpipeline.SignalProfiles)] = &PipelineConfig{
					Receivers:  []component.ID{component.MustNewID("nop")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewID("nop")},
				}
				return cfg
			},
			expected: errors.New(`pipeline "profiles": profiling signal support is at alpha level, gated under the "service.profilesSupport" feature gate`),
		},
		{
			name: "enabled-featuregate-profiles",
			cfgFn: func(t *testing.T) Config {
				require.NoError(t, featuregate.GlobalRegistry().Set(serviceProfileSupportGateID, true))

				cfg := generateConfig(t)
				cfg[pipeline.NewID(xpipeline.SignalProfiles)] = &PipelineConfig{
					Receivers:  []component.ID{component.MustNewID("nop")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewID("nop")},
				}
				return cfg
			},
			expected: nil,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.cfgFn(t)
			assert.Equal(t, tt.expected, cfg.Validate())

			// Clean up the profiles support gate, which may have been enabled in `cfgFn`.
			require.NoError(t, featuregate.GlobalRegistry().Set(serviceProfileSupportGateID, false))
		})
	}
}

func generateConfig(t *testing.T) Config {
	t.Helper()

	return map[pipeline.ID]*PipelineConfig{
		pipeline.NewID(pipeline.SignalTraces): {
			Receivers:  []component.ID{component.MustNewID("nop")},
			Processors: []component.ID{component.MustNewID("nop")},
			Exporters:  []component.ID{component.MustNewID("nop")},
		},
	}
}
