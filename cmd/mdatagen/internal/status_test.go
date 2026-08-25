// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDistroURL(t *testing.T) {
	tests := []struct {
		input  string
		output string
	}{
		{
			input:  "core",
			output: "https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol",
		},
		{
			input:  "contrib",
			output: "https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol-contrib",
		},
		{
			input:  "k8s",
			output: "https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol-k8s",
		},
		{
			input:  "otlp",
			output: "https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol-otlp",
		},
		{
			input:  "not_found",
			output: "",
		},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			assert.Equal(t, test.output, distroURL(test.input))
		})
	}
}

func TestSortedDistributions(t *testing.T) {
	tests := []struct {
		name   string
		s      Status
		result []string
	}{
		{
			"all combined",
			Status{Distributions: []string{"arm", "contrib", "core", "foo", "bar"}},
			[]string{"core", "contrib", "arm", "bar", "foo"},
		},
		{
			"core only",
			Status{Distributions: []string{"core"}},
			[]string{"core"},
		},
		{
			"core and contrib only",
			Status{Distributions: []string{"core", "contrib"}},
			[]string{"core", "contrib"},
		},
		{
			"core and contrib reversed",
			Status{Distributions: []string{"contrib", "core"}},
			[]string{"core", "contrib"},
		},
		{
			"neither core nor contrib",
			Status{Distributions: []string{"foo", "bar"}},
			[]string{"bar", "foo"},
		},
		{
			"no core, contrib, something else",
			Status{Distributions: []string{"foo", "contrib", "bar"}},
			[]string{"contrib", "bar", "foo"},
		},
		{
			"core, no contrib, something else",
			Status{Distributions: []string{"foo", "core", "bar"}},
			[]string{"core", "bar", "foo"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.result, test.s.SortedDistributions())
		})
	}
}
