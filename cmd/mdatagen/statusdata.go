// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"sort"
	"strings"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
)

// distros is a collection of distributions that can be referenced in the metadata.yaml files.
// The rules below apply to every distribution added to this list:
// - The distribution is open source and maintained by the OpenTelemetry project.
// - The link must point to a publicly accessible repository.
var distros = map[string]string{
	"core":    "https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol",
	"contrib": "https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol-contrib",
}

type Codeowners struct {
	// Active codeowners
	Active []string `mapstructure:"active"`
	// Emeritus codeowners
	Emeritus []string `mapstructure:"emeritus"`
	// Whether new codeowners are being sought
	SeekingNew bool `mapstructure:"seeking_new"`
}

type StabilityMap map[component.StabilityLevel][]string

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
