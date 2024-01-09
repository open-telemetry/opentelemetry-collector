// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"sort"
)

// distros is a collection of distributions that can be referenced in the metadata.yaml files.
// The rules below apply to every distribution added to this list:
// - The distribution must be open source.
// - The link must point to a publicly accessible repository.
var distros = map[string]string{
	"core":     "https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol",
	"contrib":  "https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol-contrib",
	"aws":      "https://github.com/aws-observability/aws-otel-collector",
	"grafana":  "https://github.com/grafana/agent",
	"observiq": "https://github.com/observIQ/observiq-otel-collector",
	"redhat":   "https://github.com/os-observability/redhat-opentelemetry-collector",
	"splunk":   "https://github.com/signalfx/splunk-otel-collector",
	"sumo":     "https://github.com/SumoLogic/sumologic-otel-collector",
	"liatrio":  "https://github.com/liatrio/liatrio-otel-collector",
}

type Codeowners struct {
	// Active codeowners
	Active []string `mapstructure:"active"`
	// Emeritus codeowners
	Emeritus []string `mapstructure:"emeritus"`
}

type Status struct {
	Stability     map[string][]string `mapstructure:"stability"`
	Distributions []string            `mapstructure:"distributions"`
	Class         string              `mapstructure:"class"`
	Warnings      []string            `mapstructure:"warnings"`
	Codeowners    *Codeowners         `mapstructure:"codeowners"`
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
