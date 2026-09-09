// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package test08

import "time"

type TimeTypeFieldConfig struct {
	StartAt        time.Time                `mapstructure:"start_at"`
	Timeout        time.Duration            `mapstructure:"timeout"`
	RetryIntervals []time.Duration          `mapstructure:"retry_intervals"`
	Durations      map[string]time.Duration `mapstructure:"durations"`
	Probes         []*time.Time             `mapstructure:"probes"`
	Window         struct {
		Begin *time.Time    `mapstructure:"begin"`
		End   time.Time     `mapstructure:"end"`
		Grace time.Duration `mapstructure:"grace"`
	} `mapstructure:"window"`
	Schedule         Schedule  `mapstructure:"schedule"`
	OptionalSchedule *Schedule `mapstructure:"optional_schedule"`
}

type Schedule struct {
	NextRun    time.Time     `mapstructure:"next_run"`
	Interval   time.Duration `mapstructure:"interval"`
	Timestamps []time.Time   `mapstructure:"timestamps"`
}
