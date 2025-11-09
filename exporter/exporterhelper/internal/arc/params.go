// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package arc // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/arc"

import (
	"time"
)

// Config exposes the knobs required by exporterhelper queue integration.
// Ranges (validated by NewController):
// - InitialLimit >= 1
// - MaxConcurrency >= InitialLimit
// - DecreaseRatio in (0, 1)
// - EwmaAlpha in (0, 1)
// - DeviationScale >= 0
//
// ControlPeriod bounds (derived): [minPeriod, maxPeriod] where
// minPeriod = 50ms, maxPeriod = 2s.
// The target period is derived from the robust EWMA mean but clamped to
// these bounds to avoid pathological long/short windows.

type Config struct {
	Enabled        bool    `mapstructure:"enabled"`
	InitialLimit   int     `mapstructure:"initial_concurrency"`
	MaxConcurrency int     `mapstructure:"max_concurrency"`
	DecreaseRatio  float64 `mapstructure:"decrease_ratio"`
	EwmaAlpha      float64 `mapstructure:"ewma_alpha"`
	DeviationScale float64 `mapstructure:"rtt_deviation_scale"`
}

func DefaultConfig() Config {
	return Config{
		Enabled:        false,
		InitialLimit:   1,
		MaxConcurrency: 200,
		DecreaseRatio:  0.9,
		EwmaAlpha:      0.4,
		DeviationScale: 2.5,
	}
}

// control period clamps
const (
	minPeriod = 50 * time.Millisecond
	maxPeriod = 2 * time.Second
)
