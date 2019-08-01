// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tailsampling

import (
	"time"

	"github.com/open-telemetry/opentelemetry-service/config/configmodels"
)

// PolicyType indicates the type of sampling policy.
type PolicyType string

const (
	// AlwaysSample samples all traces, typically used for debugging.
	AlwaysSample PolicyType = "always-sample"
	// NumericAttributeFilter sample traces that have a given numeric attribute in a specified
	// range, e.g.: attribute "http.status_code" >= 399 and <= 999.
	NumericAttributeFilter PolicyType = "numeric-attribute-filter"
	// StringAttributeFilter sample traces that a attribute, of type string, matching
	// one of the listed values.
	StringAttributeFilter PolicyType = "string-attribute-filter"
	// RateLimiting allows all traces until the specified limits are satisfied.
	RateLimiting PolicyType = "rate-limiting"
)

// PolicyCfg holds the common configuration to all policies.
type PolicyCfg struct {
	// Name given to the instance of the policy to make easy to identify it in metrics and logs.
	Name string
	// Type of the policy this will be used to match the proper configuration of the policy.
	Type PolicyType
	// Exporters hold the name of the exporters that the policy evaluator uses to make decisions
	// about whether or not sending the traces.
	Exporters []string
	// Configuration holds the settings specific to the policy.
	Configuration interface{}
}

// NumericAttributeFilterCfg holds the configurable settings to create a numeric attribute filter
// sampling policy evaluator.
type NumericAttributeFilterCfg struct {
	// Tag that the filter is going to be matching against.
	Key string `mapstructure:"key"`
	// MinValue is the minimum value of the attribute to be considered a match.
	MinValue int64 `mapstructure:"min-value"`
	// MaxValue is the maximum value of the attribute to be considered a match.
	MaxValue int64 `mapstructure:"max-value"`
}

// StringAttributeFilterCfg holds the configurable settings to create a string attribute filter
// sampling policy evaluator.
type StringAttributeFilterCfg struct {
	// Tag that the filter is going to be matching against.
	Key string `mapstructure:"key"`
	// Values is the set of attribute values that if any is equal to the actual attribute value to be considered a match.
	Values []string `mapstructure:"values"`
}

// Config holds the configuration for tail-based sampling.
type Config struct {
	configmodels.ProcessorSettings `mapstructure:",squash"`
	// DecisionWait is the desired wait time from the arrival of the first span of
	// trace until the decision about sampling it or not is evaluated.
	DecisionWait time.Duration `mapstructure:"decision-wait"`
	// NumTraces is the number of traces kept on memory. Typically most of the data
	// of a trace is released after a sampling decision is taken.
	NumTraces uint64 `mapstructure:"num-traces"`
	// ExpectedNewTracesPerSec sets the expected number of new traces sending to the tail sampling processor
	// per second. This helps with allocating data structures with closer to actual usage size.
	ExpectedNewTracesPerSec uint64 `mapstructure:"expected-new-traces-per-sec"`
}
