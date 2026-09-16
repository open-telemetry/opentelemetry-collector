// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/mdatagen/internal"

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"golang.org/x/tools/cover"

	"go.opentelemetry.io/collector/component"
)

func highestStability(ms StabilityMap) component.StabilityLevel {
	if len(ms) == 0 {
		return component.StabilityLevelUndefined
	}
	return slices.Max(slices.Collect(maps.Keys(ms)))
}

// checkCoverage in profilePath and error if it is below the target for its highest stability level.
func checkCoverage(md Metadata, targets map[component.StabilityLevel]float64, profilePath string) error {
	if md.Status == nil || slices.Contains(nonComponents, md.Status.Class) {
		return nil
	}

	level := highestStability(md.Status.Stability)
	target, ok := targets[level]
	if !ok {
		return nil
	}

	profiles, err := cover.ParseProfiles(profilePath)
	if err != nil {
		return fmt.Errorf("failed parsing coverage profile %v: %w", profilePath, err)
	}

	// Same as:
	// https://cs.opensource.google/go/go/+/refs/tags/go1.26.5:src/cmd/cover/func.go;l=149
	prefix := md.PackageName + "/"
	var covered, total int64
	for _, p := range profiles {
		if p.FileName != md.PackageName && !strings.HasPrefix(p.FileName, prefix) {
			continue
		}
		for _, b := range p.Blocks {
			total += int64(b.NumStmt)
			if b.Count > 0 {
				covered += int64(b.NumStmt)
			}
		}
	}
	if total == 0 {
		return fmt.Errorf("no coverage data for %v in %v", md.PackageName, profilePath)
	}

	pct := 100 * float64(covered) / float64(total)
	if pct < target {
		return fmt.Errorf("coverage %.1f%% for %v is below the %.1f%% target for %v", pct, md.PackageName, target, level)
	}
	return nil
}
