// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xconfmap

import (
	"errors"
	"fmt"
	"time"
)

// Config represents the receiver config settings within the collector's config.yaml
type Config struct {
	Interval       time.Duration `mapstructure:"interval"`
	NumberOfTraces int           `mapstructure:"number_of_traces"`
}

// Validate checks if the receiver configuration is valid
// this function is automatically called by the collector when it loads the configurations
// for a component
func (cfg *Config) Validate() error {
	if cfg.Interval.Minutes() < 1 {
		return errors.New("when defined, the interval has to be set to at least 1 minute (1m)")
	}

	if cfg.NumberOfTraces < 1 {
		return errors.New("number_of_traces must be greater or equal to 1")
	}
	return nil
}

// Example usage validated configuration
func Example() {
	// invalid number of traces
	myCfg := Config{
		Interval:       time.Minute,
		NumberOfTraces: 0,
	}
	err := myCfg.Validate()
	fmt.Println(err)

	// invalid interval
	myCfg = Config{
		Interval:       time.Second,
		NumberOfTraces: 1,
	}
	err = myCfg.Validate()
	fmt.Println(err)

	// valid config
	myCfg = Config{
		Interval:       time.Minute,
		NumberOfTraces: 1,
	}
	err = myCfg.Validate()
	fmt.Println(err)

	// Output:
	// number_of_traces must be greater or equal to 1
	// when defined, the interval has to be set to at least 1 minute (1m)
	// <nil>
}
