// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package fromconfig // import "go.opentelemetry.io/collector/cmd/builder/internal/fromconfig"

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"

	"golang.org/x/mod/modfile"
)

const (
	otelcolModule = corePrefix + "otelcol"
	// betaMarker is a module of the beta (v0) series required by otelcol.
	betaMarker = corePrefix + "component/componentstatus"
	// stableMarker is a module of the stable (v1) series required by otelcol.
	stableMarker = corePrefix + "component"
)

// Versions holds the versions of the two series published by a Collector release.
type Versions struct {
	Beta   string
	Stable string
}

// VersionsFunc returns the versions of a Collector release.
type VersionsFunc func(ctx context.Context, release string) (Versions, error)

// goVersions is the VersionsFunc reading the go.mod of the otelcol module at
// the given release, which requires one module of each series.
func goVersions(ctx context.Context, release string) (Versions, error) {
	out, err := runGo(ctx, "mod", "download", "-json", otelcolModule+"@"+release)
	if err != nil {
		return Versions{}, fmt.Errorf("failed to download %s@%s: %w", otelcolModule, release, err)
	}
	var info struct{ GoMod string }
	if err = json.Unmarshal(out, &info); err != nil {
		return Versions{}, fmt.Errorf("failed to decode go mod download output: %w", err)
	}
	data, err := os.ReadFile(info.GoMod)
	if err != nil {
		return Versions{}, err
	}
	return parseVersions(info.GoMod, data)
}

// parseVersions reads the versions of both series from the go.mod of otelcol.
func parseVersions(name string, gomod []byte) (Versions, error) {
	f, err := modfile.Parse(name, gomod, nil)
	if err != nil {
		return Versions{}, fmt.Errorf("failed to parse %s: %w", name, err)
	}
	var v Versions
	for _, r := range f.Require {
		switch r.Mod.Path {
		case betaMarker:
			v.Beta = r.Mod.Version
		case stableMarker:
			v.Stable = r.Mod.Version
		}
	}
	if v.Beta == "" || v.Stable == "" {
		return Versions{}, fmt.Errorf("%s does not require both %s and %s", otelcolModule, betaMarker, stableMarker)
	}
	return v, nil
}

// selectVersion returns the version of the release the module is published at,
// preferring the beta series. Each module belongs to exactly one series.
func (v Versions) selectVersion(published []string) (string, bool) {
	for _, candidate := range []string{v.Beta, v.Stable} {
		if slices.Contains(published, candidate) {
			return candidate, true
		}
	}
	return "", false
}
