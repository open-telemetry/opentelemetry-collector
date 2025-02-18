// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/mdatagen/internal"

import (
	"errors"
	"fmt"
	"sort"

	"go.opentelemetry.io/collector/component"
)

// distroURL returns the collection of distributions that can be referenced in the metadata.yaml files.
// The rules below apply to every distribution added to this list:
// - The distribution is open source and maintained by the OpenTelemetry project.
// - The link must point to a publicly accessible repository.
func distroURL(name string) string {
	switch name {
	case "core":
		return "https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol"
	case "contrib":
		return "https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol-contrib"
	case "k8s":
		return "https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol-k8s"
	case "otlp":
		return "https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol-otlp"
	default:
		return ""
	}
}

type Codeowners struct {
	// Active codeowners
	Active []string `mapstructure:"active"`
	// Emeritus codeowners
	Emeritus []string `mapstructure:"emeritus"`
	// Whether new codeowners are being sought
	SeekingNew bool `mapstructure:"seeking_new"`
}

type Status struct {
	Stability            StabilityMap `mapstructure:"stability"`
	Distributions        []string     `mapstructure:"distributions"`
	Class                string       `mapstructure:"class"`
	Warnings             []string     `mapstructure:"warnings"`
	Codeowners           *Codeowners  `mapstructure:"codeowners"`
	UnsupportedPlatforms []string     `mapstructure:"unsupported_platforms"`
}

func (s *Status) SortedDistributions() []string {
	sorted := s.Distributions
	sort.Slice(sorted, func(i, j int) bool {
		if s.Distributions[i] == "core" {
			return true
		}
		if s.Distributions[i] == "contrib" {
			return s.Distributions[j] != "core"
		}
		if s.Distributions[j] == "core" {
			return false
		}
		if s.Distributions[j] == "contrib" {
			return s.Distributions[i] == "core"
		}
		return s.Distributions[i] < s.Distributions[j]
	})
	return sorted
}

func (s *Status) Validate() error {
	var errs error
	if s == nil {
		return errors.New("missing status")
	}
	if err := s.validateClass(); err != nil {
		errs = errors.Join(errs, err)
	}

	if err := s.Stability.Validate(); err != nil {
		errs = errors.Join(errs, err)
	}
	return errs
}

func (s *Status) validateClass() error {
	if s.Class == "" {
		return errors.New("missing class")
	}
	if s.Class != "receiver" && s.Class != "processor" &&
		s.Class != "exporter" && s.Class != "connector" &&
		s.Class != "extension" && s.Class != "scraper" &&
		s.Class != "cmd" && s.Class != "pkg" {
		return fmt.Errorf("invalid class: %v", s.Class)
	}
	return nil
}

type StabilityMap map[component.StabilityLevel][]string

func (ms StabilityMap) Validate() error {
	var errs error
	if len(ms) == 0 {
		return errors.New("missing stability")
	}
	for stability, cmps := range ms {
		if len(cmps) == 0 {
			errs = errors.Join(errs, fmt.Errorf("missing component for stability: %v", stability))
		}
		for _, c := range cmps {
			if c != "metrics" &&
				c != "traces" &&
				c != "logs" &&
				c != "profiles" &&
				c != "traces_to_traces" &&
				c != "traces_to_metrics" &&
				c != "traces_to_logs" &&
				c != "traces_to_profiles" &&
				c != "metrics_to_traces" &&
				c != "metrics_to_metrics" &&
				c != "metrics_to_logs" &&
				c != "metrics_to_profiles" &&
				c != "logs_to_traces" &&
				c != "logs_to_metrics" &&
				c != "logs_to_logs" &&
				c != "logs_to_profiles" &&
				c != "profiles_to_profiles" &&
				c != "profiles_to_traces" &&
				c != "profiles_to_metrics" &&
				c != "profiles_to_logs" &&
				c != "extension" {
				errs = errors.Join(errs, fmt.Errorf("invalid component: %v", c))
			}
		}
	}
	return errs
}
