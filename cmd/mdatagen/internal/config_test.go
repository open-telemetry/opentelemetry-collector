// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureWarnings redirects warnings to a buffer for the test.
func captureWarnings(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	old := warnWriter
	warnWriter = buf
	t.Cleanup(func() { warnWriter = old })
	return buf
}

func TestLoadCentralConfig(t *testing.T) {
	t.Run("no file returns empty config", func(t *testing.T) {
		dir := t.TempDir()
		// .git bounds the walk-up to the temp dir.
		require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0o700))

		cfg, err := loadCentralConfig(dir)
		require.NoError(t, err)
		assert.Equal(t, &CentralConfig{}, cfg)
	})

	t.Run("empty file returns empty config", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, centralConfigFileName), nil, 0o600))

		cfg, err := loadCentralConfig(dir)
		require.NoError(t, err)
		assert.Equal(t, &CentralConfig{}, cfg)
	})

	t.Run("valid file is parsed", func(t *testing.T) {
		dir := t.TempDir()
		contents := `
exclusions:
  - component: go.opentelemetry.io/collector/receiver/foo
    lifecycle_test:
      enabled: false
    shutdown_test:
      enabled: false
    goleak:
      enabled: false
  - component: go.opentelemetry.io/collector/service
    feature_gates:
      - name: service.gateOne
        validation:
          enabled: false
      - name: service.gateTwo
        validation:
          enabled: false
`
		require.NoError(t, os.WriteFile(filepath.Join(dir, centralConfigFileName), []byte(contents), 0o600))

		cfg, err := loadCentralConfig(dir)
		require.NoError(t, err)
		assert.Equal(t, &CentralConfig{
			Exclusions: []ComponentExclusion{
				{
					Component:     "go.opentelemetry.io/collector/receiver/foo",
					LifecycleTest: Toggle{Enabled: new(false)},
					ShutdownTest:  Toggle{Enabled: new(false)},
					Goleak:        Toggle{Enabled: new(false)},
				},
				{
					Component: "go.opentelemetry.io/collector/service",
					FeatureGates: []FeatureGateExclusion{
						{Name: "service.gateOne", Validation: Toggle{Enabled: new(false)}},
						{Name: "service.gateTwo", Validation: Toggle{Enabled: new(false)}},
					},
				},
			},
		}, cfg)
	})

	t.Run("unknown field is rejected", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, centralConfigFileName), []byte("not_a_field: true\n"), 0o600))

		_, err := loadCentralConfig(dir)
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed parsing central config")
	})

	t.Run("unknown nested field is rejected", func(t *testing.T) {
		dir := t.TempDir()
		contents := "exclusions:\n  - component: foo\n    lifecycle_test:\n      not_a_field: true\n"
		require.NoError(t, os.WriteFile(filepath.Join(dir, centralConfigFileName), []byte(contents), 0o600))

		_, err := loadCentralConfig(dir)
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed parsing central config")
	})

	t.Run("invalid yaml is rejected", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, centralConfigFileName), []byte("exclusions: [unterminated\n"), 0o600))

		_, err := loadCentralConfig(dir)
		require.Error(t, err)
	})
}

func TestFindCentralConfig(t *testing.T) {
	t.Run("nearest config wins", func(t *testing.T) {
		root := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(root, ".git"), 0o700))
		rootCfg := filepath.Join(root, centralConfigFileName)
		require.NoError(t, os.WriteFile(rootCfg, nil, 0o600))

		sub := filepath.Join(root, "a", "b")
		require.NoError(t, os.MkdirAll(sub, 0o700))
		subCfg := filepath.Join(root, "a", centralConfigFileName)
		require.NoError(t, os.WriteFile(subCfg, nil, 0o600))

		got, err := findCentralConfig(sub)
		require.NoError(t, err)
		assert.Equal(t, subCfg, got)
	})

	t.Run("found at repo root", func(t *testing.T) {
		root := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(root, ".git"), 0o700))
		rootCfg := filepath.Join(root, centralConfigFileName)
		require.NoError(t, os.WriteFile(rootCfg, nil, 0o600))

		sub := filepath.Join(root, "a", "b")
		require.NoError(t, os.MkdirAll(sub, 0o700))

		got, err := findCentralConfig(sub)
		require.NoError(t, err)
		assert.Equal(t, rootCfg, got)
	})

	t.Run("search does not escape the repository root", func(t *testing.T) {
		outer := t.TempDir()
		// A config above the repo root must NOT be discovered.
		require.NoError(t, os.WriteFile(filepath.Join(outer, centralConfigFileName), nil, 0o600))

		repo := filepath.Join(outer, "repo")
		sub := filepath.Join(repo, "component")
		require.NoError(t, os.MkdirAll(sub, 0o700))
		require.NoError(t, os.Mkdir(filepath.Join(repo, ".git"), 0o700))

		got, err := findCentralConfig(sub)
		require.NoError(t, err)
		assert.Empty(t, got)
	})
}

func TestCentralConfigApplyTo(t *testing.T) {
	const pkg = "go.opentelemetry.io/collector/receiver/foo"

	t.Run("central config sets skips additively", func(t *testing.T) {
		md := Metadata{
			PackageName: pkg,
			FeatureGates: []FeatureGate{
				{ID: "foo.gateA"},
				{ID: "foo.gateB"},
			},
		}
		cfg := &CentralConfig{
			Exclusions: []ComponentExclusion{
				{
					Component:     pkg,
					LifecycleTest: Toggle{Enabled: new(false)},
					ShutdownTest:  Toggle{Enabled: new(false)},
					Goleak:        Toggle{Enabled: new(false)},
					FeatureGates: []FeatureGateExclusion{
						{Name: "foo.gateA", Validation: Toggle{Enabled: new(false)}},
					},
				},
			},
		}
		cfg.applyTo(&md)

		assert.True(t, md.Tests.SkipLifecycle)
		assert.True(t, md.Tests.SkipShutdown)
		assert.True(t, md.Tests.GoLeak.Skip)
		assert.True(t, md.FeatureGates[0].SkipStrictValidation)
		assert.False(t, md.FeatureGates[1].SkipStrictValidation)
	})

	t.Run("metadata skips are preserved when central config is empty", func(t *testing.T) {
		md := Metadata{
			PackageName: pkg,
			Tests: Tests{
				SkipLifecycle: true,
				SkipShutdown:  true,
				GoLeak:        GoLeak{Skip: true},
			},
			FeatureGates: []FeatureGate{{ID: "foo.gateA", SkipStrictValidation: true}},
		}
		cfg := &CentralConfig{}
		cfg.applyTo(&md)

		assert.True(t, md.Tests.SkipLifecycle)
		assert.True(t, md.Tests.SkipShutdown)
		assert.True(t, md.Tests.GoLeak.Skip)
		assert.True(t, md.FeatureGates[0].SkipStrictValidation)
	})

	t.Run("gate is exempted only by an explicit enabled: false", func(t *testing.T) {
		md := Metadata{
			PackageName: pkg,
			FeatureGates: []FeatureGate{
				{ID: "foo.absent"},
				{ID: "foo.enabled"},
			},
		}
		cfg := &CentralConfig{
			Exclusions: []ComponentExclusion{
				{Component: pkg, FeatureGates: []FeatureGateExclusion{
					{Name: "foo.absent"}, // no enabled key
					{Name: "foo.enabled", Validation: Toggle{Enabled: new(true)}}, // explicitly enabled
				}},
			},
		}
		cfg.applyTo(&md)

		assert.False(t, md.FeatureGates[0].SkipStrictValidation)
		assert.False(t, md.FeatureGates[1].SkipStrictValidation)
	})

	t.Run("all entries matching a component apply", func(t *testing.T) {
		md := Metadata{
			PackageName:  pkg,
			FeatureGates: []FeatureGate{{ID: "foo.gateA"}},
		}
		cfg := &CentralConfig{
			Exclusions: []ComponentExclusion{
				{Component: pkg, LifecycleTest: Toggle{Enabled: new(false)}},
				{Component: pkg, FeatureGates: []FeatureGateExclusion{{Name: "foo.gateA", Validation: Toggle{Enabled: new(false)}}}},
			},
		}
		cfg.applyTo(&md)

		assert.True(t, md.Tests.SkipLifecycle)
		assert.True(t, md.FeatureGates[0].SkipStrictValidation)
	})

	t.Run("other components are unaffected", func(t *testing.T) {
		md := Metadata{
			PackageName:  "go.opentelemetry.io/collector/receiver/other",
			FeatureGates: []FeatureGate{{ID: "foo.gateA"}},
		}
		cfg := &CentralConfig{
			Exclusions: []ComponentExclusion{
				{
					Component:     pkg,
					LifecycleTest: Toggle{Enabled: new(false)},
					FeatureGates:  []FeatureGateExclusion{{Name: "foo.gateA", Validation: Toggle{Enabled: new(false)}}},
				},
			},
		}
		cfg.applyTo(&md)

		assert.False(t, md.Tests.SkipLifecycle)
		assert.False(t, md.FeatureGates[0].SkipStrictValidation)
	})
}

// TestLoadMetadataAppliesCentralConfig checks that a gate failing strict
// validation passes once the central config grandfathers it.
func TestLoadMetadataAppliesCentralConfig(t *testing.T) {
	const metadataYAML = `type: sample
status:
  class: receiver
  stability:
    beta: [metrics]

feature_gates:
  - id: receiver.sample.grandfathered
    description: 'grandfathered gate'
    stage: alpha
    from_version: 'v0.100.0'
    reference_url: 'https://example.com/not-a-github-issue'
`

	// The module's import path resolves to "e2etest"; .git bounds discovery to it.
	setup := func(t *testing.T, withCentral bool) string {
		dir := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0o700))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module e2etest\n"), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "component.go"), []byte("package e2etest\n"), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "metadata.yaml"), []byte(metadataYAML), 0o600))
		if withCentral {
			central := "exclusions:\n  - component: e2etest\n    feature_gates:\n      - name: receiver.sample.grandfathered\n        validation:\n          enabled: false\n"
			require.NoError(t, os.WriteFile(filepath.Join(dir, centralConfigFileName), []byte(central), 0o600))
		}
		return filepath.Join(dir, "metadata.yaml")
	}

	t.Run("gate fails strict validation without central config", func(t *testing.T) {
		_, err := LoadMetadata(setup(t, false))
		require.Error(t, err)
		assert.ErrorContains(t, err, "must be a GitHub issue URL")
	})

	t.Run("central config grandfathers the gate without a deprecation warning", func(t *testing.T) {
		warnings := captureWarnings(t)
		md, err := LoadMetadata(setup(t, true))
		require.NoError(t, err)
		require.Len(t, md.FeatureGates, 1)
		assert.True(t, md.FeatureGates[0].SkipStrictValidation)
		// Central config is the recommended replacement; no warning.
		assert.Empty(t, warnings.String())
	})
}

// TestLoadMetadataWarnsOnDeprecatedSkipStrictValidation checks that setting the
// flag in metadata.yaml warns, even for a gate that already passes validation.
func TestLoadMetadataWarnsOnDeprecatedSkipStrictValidation(t *testing.T) {
	const metadataYAML = `type: sample
status:
  class: receiver
  stability:
    beta: [metrics]

feature_gates:
  - id: receiver.sample.validgate
    description: 'redundantly sets the deprecated flag'
    stage: alpha
    from_version: 'v0.100.0'
    reference_url: 'https://github.com/open-telemetry/opentelemetry-collector/issues/12345'
    skip_strict_validation: true
`
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module deprtest\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "component.go"), []byte("package deprtest\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "metadata.yaml"), []byte(metadataYAML), 0o600))

	warnings := captureWarnings(t)
	_, err := LoadMetadata(filepath.Join(dir, "metadata.yaml"))
	require.NoError(t, err)
	assert.Contains(t, warnings.String(), `feature gate "receiver.sample.validgate" sets deprecated 'skip_strict_validation'`)
}
