// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package fromconfig // import "go.opentelemetry.io/collector/cmd/builder/internal/fromconfig"

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// sentinelModule is looked up with the first batch to tell apart "the module
// does not exist" from "module lookups are unavailable".
const sentinelModule = corePrefix + "receiver/otlpreceiver"

// maxArgBytes bounds the size of the module arguments of a single `go list`
// invocation, to stay well within the Windows command-line limit.
const maxArgBytes = 6000

// ListedModule is the subset of `go list -m -json` output used for verification.
type ListedModule struct {
	Path     string
	Versions []string
	Error    *struct{ Err string }
}

// exists reports whether the module was found, with at least one version.
func (m ListedModule) exists() bool {
	return m.Error == nil && len(m.Versions) > 0
}

// ListFunc looks up modules and their available versions. Modules that cannot
// be found are reported without versions, or with an Error, not as a failure.
type ListFunc func(ctx context.Context, modules []string) ([]ListedModule, error)

// errLookupUnavailable is returned when module lookups do not work at all.
var errLookupUnavailable = errors.New("module lookup unavailable")

// selection is the module selected for a component, with the version chosen for it.
type selection struct {
	module  ListedModule
	version string
}

// verified holds the outcome of verifying the candidates of a set of components.
type verified struct {
	// modules holds, for each resolved component, the selected module.
	modules map[Component]selection
	// found holds the components for which at least one candidate module exists
	// but none was accepted.
	found map[Component]bool
}

// verify looks up the candidate modules of the components through list, one
// priority level at a time, and selects for each component the highest-priority
// module that exists and for which choose returns a version.
func verify(ctx context.Context, list ListFunc, components []Component, choose func(ListedModule) (string, bool)) (verified, error) {
	result := verified{
		modules: map[Component]selection{},
		found:   map[Component]bool{},
	}

	cands := make(map[Component][]candidate, len(components))
	var levels []int
	for _, c := range components {
		cs := candidates(c)
		cands[c] = cs
		for _, cand := range cs {
			if !slices.Contains(levels, cand.level) {
				levels = append(levels, cand.level)
			}
		}
	}
	slices.Sort(levels)

	needSentinel := len(levels) > 0
	for _, level := range levels {
		paths, byPath := levelPaths(level, components, cands, result.modules)
		if len(paths) == 0 {
			continue
		}

		if needSentinel && !slices.Contains(paths, sentinelModule) {
			paths = append([]string{sentinelModule}, paths...)
		}

		for i, chunk := range chunkArgs(paths) {
			listed, err := list(ctx, chunk)
			if err != nil {
				return verified{}, fmt.Errorf("failed to verify component modules: %w", err)
			}
			if needSentinel && i == 0 {
				needSentinel = false
				if err := checkSentinel(listed); err != nil {
					return verified{}, err
				}
			}
			for _, m := range listed {
				if !m.exists() {
					continue
				}
				for _, c := range byPath[m.Path] {
					if _, done := result.modules[c]; done {
						continue
					}
					if version, ok := choose(m); ok {
						result.modules[c] = selection{module: m, version: version}
					} else {
						result.found[c] = true
					}
				}
			}
		}
	}
	return result, nil
}

// levelPaths returns the distinct module paths to look up for the candidates of
// the given level, and the components each path is a candidate for. Components
// that are already resolved are skipped.
func levelPaths(level int, components []Component, cands map[Component][]candidate, resolved map[Component]selection) ([]string, map[string][]Component) {
	byPath := map[string][]Component{}
	var paths []string
	for _, c := range components {
		if _, done := resolved[c]; done {
			continue
		}
		// Each component has at most one candidate per level.
		for _, cand := range cands[c] {
			if cand.level != level {
				continue
			}
			if _, ok := byPath[cand.path]; !ok {
				paths = append(paths, cand.path)
			}
			byPath[cand.path] = append(byPath[cand.path], c)
		}
	}
	return paths, byPath
}

// checkSentinel fails if the sentinel module is missing from the first batch,
// which means module lookups do not work at all.
func checkSentinel(listed []ListedModule) error {
	for _, m := range listed {
		if m.Path != sentinelModule {
			continue
		}
		if m.exists() {
			return nil
		}
		reason := "module not found"
		if m.Error != nil {
			reason = m.Error.Err
		}
		return fmt.Errorf("failed to verify component modules: %w: %s\n"+
			"ocb init --from-config needs access to the Go module proxy; run `ocb init` without --from-config for a default manifest",
			errLookupUnavailable, firstLine(reason))
	}
	return fmt.Errorf("failed to verify component modules: %w: no result for %s", errLookupUnavailable, sentinelModule)
}

// firstLine returns the first line of s.
func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return line
}

// chunkArgs splits args in groups whose total size stays below maxArgBytes.
func chunkArgs(args []string) [][]string {
	var chunks [][]string
	var current []string
	size := 0
	for _, a := range args {
		if len(current) > 0 && size+len(a)+1 > maxArgBytes {
			chunks = append(chunks, current)
			current, size = nil, 0
		}
		current = append(current, a)
		size += len(a) + 1
	}
	if len(current) > 0 {
		chunks = append(chunks, current)
	}
	return chunks
}

// goList is the ListFunc backed by the Go toolchain.
func goList(ctx context.Context, modules []string) ([]ListedModule, error) {
	out, err := runGo(ctx, append([]string{"list", "-m", "-e", "-mod=mod", "-versions", "-json"}, modules...)...)
	if err != nil {
		return nil, err
	}
	return decodeListed(bytes.NewReader(out))
}

// newLookupModule creates a throwaway module to run go commands from.
func newLookupModule() (string, error) {
	dir, err := os.MkdirTemp("", "ocb-from-config")
	if err != nil {
		return "", err
	}
	err = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module ocb.invalid/lookup\n\ngo 1.25\n"), 0o600)
	if err != nil {
		return "", errors.Join(err, os.RemoveAll(dir))
	}
	return dir, nil
}

// runGo runs the go command from within a throwaway module, and returns its output.
func runGo(ctx context.Context, args ...string) ([]byte, error) {
	dir, err := newLookupModule()
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	cmd := exec.CommandContext(ctx, "go", args...) //nolint:gosec // G204: the arguments are built from validated module paths
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go %s failed: %w (%s)", args[0], err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

// decodeListed decodes the stream of concatenated JSON objects printed by `go list -json`.
func decodeListed(r io.Reader) ([]ListedModule, error) {
	var out []ListedModule
	dec := json.NewDecoder(r)
	for {
		var m ListedModule
		err := dec.Decode(&m)
		if errors.Is(err, io.EOF) {
			return out, nil
		}
		if err != nil {
			return nil, fmt.Errorf("failed to decode go list output: %w", err)
		}
		out = append(out, m)
	}
}
