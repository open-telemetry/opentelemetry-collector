// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/mdatagen/internal"

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
)

// Distros is a collection of distributions that can be referenced in the metadata.yaml files.
// The rules below apply to every distribution added to this list:
// - The distribution is open source and maintained by the OpenTelemetry project.
// - The link must point to a publicly accessible repository.
var Distros = map[string]string{
	"core":    "https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol",
	"contrib": "https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol-contrib",
	"k8s":     "https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol-k8s",
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
	NotComponent         bool         `mapstructure:"not_component"`
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
	if err := s.validateStability(); err != nil {
		errs = errors.Join(errs, err)
	}
	return errs
}

func (s *Status) validateClass() error {
	if s.Class == "" {
		return errors.New("missing class")
	}
	if s.Class != "receiver" && s.Class != "processor" && s.Class != "exporter" && s.Class != "connector" && s.Class != "extension" && s.Class != "cmd" && s.Class != "pkg" {
		return fmt.Errorf("invalid class: %v", s.Class)
	}
	return nil
}

func (s *Status) validateStability() error {
	var errs error
	if len(s.Stability) == 0 {
		return errors.New("missing stability")
	}
	for stability, component := range s.Stability {
		if len(component) == 0 {
			errs = errors.Join(errs, fmt.Errorf("missing component for stability: %v", stability))
		}
		for _, c := range component {
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

type StabilityMap map[component.StabilityLevel][]string

func (ms *StabilityMap) Unmarshal(parser *confmap.Conf) error {
	*ms = make(StabilityMap)
	raw := make(map[string][]string)
	err := parser.Unmarshal(&raw)
	if err != nil {
		return err
	}
	for k, v := range raw {
		switch strings.ToLower(k) {
		case strings.ToLower(component.StabilityLevelUnmaintained.String()):
			(*ms)[component.StabilityLevelUnmaintained] = v
		case strings.ToLower(component.StabilityLevelDeprecated.String()):
			(*ms)[component.StabilityLevelDeprecated] = v
		case strings.ToLower(component.StabilityLevelDevelopment.String()):
			(*ms)[component.StabilityLevelDevelopment] = v
		case strings.ToLower(component.StabilityLevelAlpha.String()):
			(*ms)[component.StabilityLevelAlpha] = v
		case strings.ToLower(component.StabilityLevelBeta.String()):
			(*ms)[component.StabilityLevelBeta] = v
		case strings.ToLower(component.StabilityLevelStable.String()):
			(*ms)[component.StabilityLevelStable] = v
		default:
			return errors.New("invalid stability level: " + k)
		}
	}
	return nil
}
