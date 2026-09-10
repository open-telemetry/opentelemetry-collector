// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
)

func writeProfile(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "cover.out")
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))
	return path
}

func TestHighestStability(t *testing.T) {
	assert.Equal(t, component.StabilityLevelUndefined, highestStability(nil))
	assert.Equal(t, component.StabilityLevelBeta, highestStability(StabilityMap{
		component.StabilityLevelBeta: []string{"traces"},
	}))
	assert.Equal(t, component.StabilityLevelStable, highestStability(StabilityMap{
		component.StabilityLevelBeta:   []string{"traces"},
		component.StabilityLevelStable: []string{"metrics"},
	}))
}

// Go coverage profile format available here:
// https://github.com/golang/tools/blob/v0.49.0/cover/profile.go#L55-L58.
func TestCheckCoverage(t *testing.T) {
	const pkg = "go.opentelemetry.io/collector/receiver/foo"

	stableMD := func() Metadata {
		return Metadata{
			PackageName: pkg,
			Status: &Status{
				Class:     "receiver",
				Stability: StabilityMap{component.StabilityLevelStable: []string{"metrics"}},
			},
		}
	}

	t.Run("below target fails", func(t *testing.T) {
		profile := writeProfile(t, "mode: atomic\n"+
			pkg+"/foo.go:1.1,10.2 8 1\n"+
			pkg+"/foo.go:11.1,20.2 2 0\n")
		err := checkCoverage(stableMD(), map[component.StabilityLevel]float64{component.StabilityLevelStable: 90}, profile)
		require.Error(t, err)
		assert.ErrorContains(t, err, "coverage 80.0% for "+pkg+" is below the 90% target for Stable")
	})

	t.Run("at target passes", func(t *testing.T) {
		profile := writeProfile(t, "mode: atomic\n"+
			pkg+"/foo.go:1.1,10.2 8 1\n"+
			pkg+"/foo.go:11.1,20.2 2 0\n")
		err := checkCoverage(stableMD(), map[component.StabilityLevel]float64{component.StabilityLevelStable: 80}, profile)
		require.NoError(t, err)
	})

	t.Run("subpackage statements are counted", func(t *testing.T) {
		profile := writeProfile(t, "mode: atomic\n"+
			pkg+"/foo.go:1.1,10.2 5 1\n"+
			pkg+"/internal/bar.go:1.1,10.2 5 1\n")
		err := checkCoverage(stableMD(), map[component.StabilityLevel]float64{component.StabilityLevelStable: 100}, profile)
		require.NoError(t, err)
	})

	t.Run("a sibling package with a shared prefix is not counted", func(t *testing.T) {
		profile := writeProfile(t, "mode: atomic\n"+
			pkg+"/foo.go:1.1,10.2 5 0\n"+
			pkg+"other/bar.go:1.1,10.2 5 1\n")
		err := checkCoverage(stableMD(), map[component.StabilityLevel]float64{component.StabilityLevelStable: 100}, profile)
		require.Error(t, err)
		assert.ErrorContains(t, err, "coverage 0.0%")
	})

	t.Run("no target for the level passes", func(t *testing.T) {
		profile := writeProfile(t, "mode: atomic\n"+pkg+"/foo.go:1.1,10.2 5 0\n")
		err := checkCoverage(stableMD(), map[component.StabilityLevel]float64{component.StabilityLevelBeta: 100}, profile)
		require.NoError(t, err)
	})

	t.Run("component absent from the profile errors", func(t *testing.T) {
		profile := writeProfile(t, "mode: atomic\n"+"go.opentelemetry.io/collector/receiver/other/foo.go:1.1,10.2 5 1\n")
		err := checkCoverage(stableMD(), map[component.StabilityLevel]float64{component.StabilityLevelStable: 10}, profile)
		require.Error(t, err)
		assert.ErrorContains(t, err, "no coverage data for "+pkg)
	})

	t.Run("nonComponents class passes without checking", func(t *testing.T) {
		md := Metadata{
			PackageName: "go.opentelemetry.io/collector/pkg/foo",
			Status:      &Status{Class: "pkg"},
		}
		profile := writeProfile(t, "mode: atomic\n"+"unrelated/foo.go:1.1,10.2 5 0\n")
		err := checkCoverage(md, map[component.StabilityLevel]float64{component.StabilityLevelStable: 100}, profile)
		require.NoError(t, err)
	})

	t.Run("missing status passes without checking", func(t *testing.T) {
		md := Metadata{PackageName: pkg}
		profile := writeProfile(t, "mode: atomic\n"+pkg+"/foo.go:1.1,10.2 5 0\n")
		err := checkCoverage(md, map[component.StabilityLevel]float64{component.StabilityLevelStable: 100}, profile)
		require.NoError(t, err)
	})

	t.Run("highest of several declared levels is applied", func(t *testing.T) {
		md := Metadata{
			PackageName: pkg,
			Status: &Status{
				Class: "receiver",
				Stability: StabilityMap{
					component.StabilityLevelBeta:   []string{"traces"},
					component.StabilityLevelStable: []string{"metrics"},
				},
			},
		}
		profile := writeProfile(t, "mode: atomic\n"+pkg+"/foo.go:1.1,10.2 5 0\n")
		err := checkCoverage(md, map[component.StabilityLevel]float64{
			component.StabilityLevelBeta:   10, // would pass
			component.StabilityLevelStable: 90, // must fail: highest level wins
		}, profile)
		require.Error(t, err)
		assert.ErrorContains(t, err, "target for Stable")
	})
}
