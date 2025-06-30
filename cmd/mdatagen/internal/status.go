// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/mdatagen/internal"

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"time"

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
	Stability            StabilityMap   `mapstructure:"stability"`
	Distributions        []string       `mapstructure:"distributions"`
	Class                string         `mapstructure:"class"`
	Warnings             []string       `mapstructure:"warnings"`
	Codeowners           *Codeowners    `mapstructure:"codeowners"`
	UnsupportedPlatforms []string       `mapstructure:"unsupported_platforms"`
	Deprecation          DeprecationMap `mapstructure:"deprecation"`
	CodeCovComponentID   string         `mapstructure:"codecov_component_id"`
	DisableCodeCov       bool           `mapstructure:"disable_codecov_badge"`
}

type DeprecationMap map[string]DeprecationInfo

type DeprecationInfo struct {
	Date      string `mapstructure:"date"`
	Migration string `mapstructure:"migration"`
}

var validClasses = []string{
	"cmd",
	"connector",
	"converter",
	"exporter",
	"extension",
	"pkg",
	"processor",
	"provider",
	"receiver",
	"scraper",
}

var validStabilityKeys = []string{
	"converter",
	"extension",
	"logs",
	"logs_to_traces",
	"logs_to_metrics",
	"logs_to_logs",
	"logs_to_profiles",
	"metrics",
	"metrics_to_traces",
	"metrics_to_metrics",
	"metrics_to_logs",
	"metrics_to_profiles",
	"profiles",
	"profiles_to_profiles",
	"profiles_to_traces",
	"profiles_to_metrics",
	"profiles_to_logs",
	"provider",
	"traces_to_traces",
	"traces_to_metrics",
	"traces_to_logs",
	"traces_to_profiles",
	"traces",
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
	if err := s.Deprecation.Validate(s.Stability); err != nil {
		errs = errors.Join(errs, err)
	}
	return errs
}

func (s *Status) validateClass() error {
	if s.Class == "" {
		return errors.New("missing class")
	}
	if !slices.Contains(validClasses, s.Class) {
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
			if !slices.Contains(validStabilityKeys, c) {
				errs = errors.Join(errs, fmt.Errorf("invalid component: %v", c))
			}
		}
	}
	return errs
}

func (dm DeprecationMap) Validate(ms StabilityMap) error {
	var errs error
	for stability, cmps := range ms {
		if stability != component.StabilityLevelDeprecated {
			continue
		}
		for _, c := range cmps {
			depInfo, found := dm[c]
			if !found {
				errs = errors.Join(errs, fmt.Errorf("deprecated component missing deprecation date and migration guide for %v", c))
				continue
			}
			if depInfo.Migration == "" {
				errs = errors.Join(errs, fmt.Errorf("deprecated component missing migration guide: %v", c))
			}
			if depInfo.Date == "" {
				errs = errors.Join(errs, fmt.Errorf("deprecated component missing date in YYYY-MM-DD format: %v", c))
			} else {
				_, err := time.Parse("2006-01-02", depInfo.Date)
				if err != nil {
					errs = errors.Join(errs, fmt.Errorf("deprecated component missing valid date in YYYY-MM-DD format: %v", c))
				}
			}
		}
	}
	return errs
}
