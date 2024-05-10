// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
