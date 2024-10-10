// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configtimeout // import "go.opentelemetry.io/collector/config/configtimeout"

import (
	"errors"
	"fmt"
	"time"
)

// DefaultTimeout is the default timeout setting used by the
// exporterhelper Timeout sender.
const DefaultTimeout = 5 * time.Second

// Policy represents a policy towards handling short timeouts, which
// are cases where an exporter component is configured for a timeout
// and the arriving context has a shorter deadline.
type Policy string

const (
	PolicySustain Policy = "sustain"
	PolicyIgnore  Policy = "ignore"
	PolicyAbort   Policy = "abort"
	policyUnset   Policy = ""

	// PolicyDefault selects the original default behavior of the
	// exporterhelper component, which is to send a request with
	// shorter deadline than configured.
	PolicyDefault = PolicySustain
)

// TimeoutConfig for timeout. The timeout applies to individual attempts to send data to the backend.
type TimeoutConfig struct {
	// Timeout is the timeout for every attempt to send data to the backend.
	// A zero timeout means no timeout.
	Timeout time.Duration `mapstructure:"timeout"`

	// Policy indicates how the exporter will handle requests that
	// arrive with a shorter deadline than the configured timeout.
	// Note that because the TimeoutConfig is traditionally
	// struct-embedded in the parent configuration, we use a
	// relatively long descriptive tag name--"timeout" does not
	// stutter as a result.
	ShortTimeoutPolicy Policy `mapstructure:"short_timeout_policy"`
}

func (ts *TimeoutConfig) Validate() error {
	// Negative timeouts are not acceptable, since all sends will fail.
	if ts.Timeout < 0 {
		return errors.New("'timeout' must be non-negative")
	}
	return nil
}

// NewDefaultConfig returns the default config for TimeoutConfig.
func NewDefaultConfig() TimeoutConfig {
	return TimeoutConfig{
		Timeout:            DefaultTimeout,
		ShortTimeoutPolicy: policyUnset,
	}
}

func (tp *Policy) Validate() error {
	switch *tp {
	case PolicySustain, PolicyIgnore, PolicyAbort, policyUnset:
		return nil
	default:
		return fmt.Errorf("unsupported 'short_timeout_policy' %v", *tp)
	}

}
