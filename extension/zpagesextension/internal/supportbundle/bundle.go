// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package supportbundle builds support bundles: zip archives of diagnostic
// files from a running collector.
package supportbundle // import "go.opentelemetry.io/collector/extension/zpagesextension/internal/supportbundle"

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"time"
)

// File is one entry in a support bundle.
type File struct {
	// Path is relative to the bundle root and uses forward slashes.
	Path    string
	Content []byte
}

// Gatherer collects a set of diagnostic files.
type Gatherer interface {
	// Name identifies the gatherer in config, in requests and in the manifest.
	Name() string

	// Gather collects the diagnostic files.
	Gather(ctx context.Context) ([]File, error)
}

// ErrUnavailable tells the bundler that a gatherer cannot run in this
// collector, for example because a runtime hook is off. Wrap it to add detail.
var ErrUnavailable = errors.New("gatherer is unavailable")

// SkipReason tells why a gatherer did not contribute to a bundle.
type SkipReason string

const (
	SkipDisabledInConfig SkipReason = "disabled_in_config"
	SkipUnknown          SkipReason = "unknown"
	SkipUnavailable      SkipReason = "unavailable"
	SkipFailed           SkipReason = "failed"
)

// Entry is a gatherer and its config state.
type Entry struct {
	Gatherer Gatherer
	Enabled  bool
}

// errNothingSelected tells that no requested gatherer is enabled and known.
var errNothingSelected = errors.New("no requested gatherer is enabled")

// errNothingGathered tells that every selected gatherer was skipped.
var errNothingGathered = errors.New("no gatherer produced content")

type gatheredFile struct {
	File
	gatherer    string
	collectedAt time.Time
}

// result holds everything one bundle request produced.
type result struct {
	selection Selection
	files     []gatheredFile
}

// gatherers holds the available gatherers in registration order and by name.
type gatherers struct {
	ordered []Entry
	byName  map[string]Entry
}

func newGatherers(entries []Entry) gatherers {
	g := gatherers{ordered: entries, byName: make(map[string]Entry, len(entries))}
	for _, e := range entries {
		g.byName[e.Gatherer.Name()] = e
	}
	return g
}

// gather runs the gatherers that the request selects. If requested is empty,
// it runs every enabled gatherer. It returns an error only when no gatherer
// contributes. The selection still holds the reasons in that case.
func gather(ctx context.Context, available gatherers, requested []string) (result, error) {
	res := result{selection: Selection{Requested: requested}}

	var selected []Gatherer
	if len(requested) == 0 {
		for _, e := range available.ordered {
			if e.Enabled {
				selected = append(selected, e.Gatherer)
				continue
			}
			res.selection.skip(e.Gatherer.Name(), SkipDisabledInConfig, nil)
		}
	} else {
		for _, name := range requested {
			e, ok := available.byName[name]
			switch {
			case !ok:
				res.selection.skip(name, SkipUnknown, nil)
			case !e.Enabled:
				res.selection.skip(name, SkipDisabledInConfig, nil)
			default:
				selected = append(selected, e.Gatherer)
			}
		}
	}
	if len(selected) == 0 {
		return res, errNothingSelected
	}

	paths := map[string]string{manifestPath: "bundle"}
	for _, g := range selected {
		name := g.Name()
		collectedAt := time.Now()
		files, err := g.Gather(ctx)
		if err == nil {
			err = checkPaths(files, paths)
		}
		switch {
		case errors.Is(err, ErrUnavailable):
			res.selection.skip(name, SkipUnavailable, err)
			continue
		case err != nil:
			res.selection.skip(name, SkipFailed, err)
			continue
		}

		for _, f := range files {
			paths[f.Path] = name
			res.files = append(res.files, gatheredFile{File: f, gatherer: name, collectedAt: collectedAt})
		}
		res.selection.Ran = append(res.selection.Ran, name)
	}
	if len(res.selection.Ran) == 0 {
		return res, errNothingGathered
	}
	return res, nil
}

// checkPaths makes sure that a gatherer's files stay inside the bundle root and
// do not replace files from other gatherers.
func checkPaths(files []File, seen map[string]string) error {
	own := make(map[string]struct{}, len(files))
	for _, f := range files {
		// fs.ValidPath rejects ".." elements. Some zip tools read a backslash
		// as a separator, so reject it too.
		if !fs.ValidPath(f.Path) || f.Path == "." || strings.Contains(f.Path, `\`) {
			return fmt.Errorf("invalid file path %q", f.Path)
		}
		if owner, ok := seen[f.Path]; ok {
			return fmt.Errorf("file path %q is already used by %q", f.Path, owner)
		}
		if _, ok := own[f.Path]; ok {
			return fmt.Errorf("file path %q is used more than once", f.Path)
		}
		own[f.Path] = struct{}{}
	}
	return nil
}
