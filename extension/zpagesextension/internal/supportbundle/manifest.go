// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package supportbundle // import "go.opentelemetry.io/collector/extension/zpagesextension/internal/supportbundle"

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"runtime"
	"time"

	"go.opentelemetry.io/collector/component"
)

const (
	manifestPath = "manifest.yaml"
	// formatVersion changes when the bundle layout or the manifest changes in
	// a way that breaks readers.
	formatVersion = "v1"
)

// Manifest describes a bundle. It is always present in the bundle.
type Manifest struct {
	Format    string            `yaml:"format"`
	BundleID  string            `yaml:"bundle_id"`
	Capture   Capture           `yaml:"capture"`
	Collector CollectorInfo     `yaml:"collector"`
	Runtime   RuntimeInfo       `yaml:"runtime"`
	Resource  map[string]string `yaml:"resource,omitempty"`
	Gatherers Selection         `yaml:"gatherers"`
	Files     []FileInfo        `yaml:"files"`
}

// Capture records when the bundle was collected.
type Capture struct {
	Start time.Time `yaml:"start"`
	End   time.Time `yaml:"end"`
}

// CollectorInfo identifies the collector build.
type CollectorInfo struct {
	Command     string `yaml:"command"`
	Description string `yaml:"description"`
	Version     string `yaml:"version"`
}

// RuntimeInfo describes the process and the Go runtime.
type RuntimeInfo struct {
	GoVersion  string `yaml:"go_version"`
	GOOS       string `yaml:"goos"`
	GOARCH     string `yaml:"goarch"`
	NumCPU     int    `yaml:"num_cpu"`
	GOMAXPROCS int    `yaml:"gomaxprocs"`
	Hostname   string `yaml:"hostname,omitempty"`
}

// Selection records which gatherers a request asked for, which ran and which
// were skipped.
type Selection struct {
	// Requested is empty when the request did not select gatherers. All
	// enabled gatherers run in that case.
	Requested []string  `yaml:"requested,omitempty"`
	Ran       []string  `yaml:"ran"`
	Skipped   []Skipped `yaml:"skipped,omitempty"`
}

// Skipped records a gatherer that did not contribute to the bundle. The JSON
// tags keep the log keys the same as the manifest keys.
type Skipped struct {
	Name   string     `yaml:"name" json:"name"`
	Reason SkipReason `yaml:"reason" json:"reason"`
	Error  string     `yaml:"error,omitempty" json:"error,omitempty"`
}

func (s *Selection) skip(name string, reason SkipReason, err error) {
	sk := Skipped{Name: name, Reason: reason}
	if err != nil {
		sk.Error = err.Error()
	}
	s.Skipped = append(s.Skipped, sk)
}

// FileInfo describes one file in the bundle.
type FileInfo struct {
	Path        string    `yaml:"path"`
	Gatherer    string    `yaml:"gatherer"`
	CollectedAt time.Time `yaml:"collected_at"`
	SizeBytes   int       `yaml:"size_bytes"`
	SHA256      string    `yaml:"sha256"`
}

func newManifest(bundleID string, start, end time.Time, buildInfo component.BuildInfo, resource map[string]string, res result) Manifest {
	// The hostname is useful but not required, so an error leaves it empty.
	hostname, _ := os.Hostname()
	m := Manifest{
		Format:   formatVersion,
		BundleID: bundleID,
		Capture:  Capture{Start: start.UTC(), End: end.UTC()},
		Collector: CollectorInfo{
			Command:     buildInfo.Command,
			Description: buildInfo.Description,
			Version:     buildInfo.Version,
		},
		Runtime: RuntimeInfo{
			GoVersion:  runtime.Version(),
			GOOS:       runtime.GOOS,
			GOARCH:     runtime.GOARCH,
			NumCPU:     runtime.NumCPU(),
			GOMAXPROCS: runtime.GOMAXPROCS(0),
			Hostname:   hostname,
		},
		Resource:  resource,
		Gatherers: res.selection,
		Files:     make([]FileInfo, 0, len(res.files)),
	}
	for _, f := range res.files {
		sum := sha256.Sum256(f.Content)
		m.Files = append(m.Files, FileInfo{
			Path:        f.Path,
			Gatherer:    f.gatherer,
			CollectedAt: f.collectedAt.UTC(),
			SizeBytes:   len(f.Content),
			SHA256:      hex.EncodeToString(sum[:]),
		})
	}
	return m
}
