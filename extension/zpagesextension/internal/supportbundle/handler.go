// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package supportbundle // import "go.opentelemetry.io/collector/extension/zpagesextension/internal/supportbundle"

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.yaml.in/yaml/v3"

	"go.opentelemetry.io/collector/component"
)

const (
	gatherersParam  = "gatherers"
	filenameTimeFmt = "20060102T150405Z"
)

// Settings holds the collector data that every manifest records.
type Settings struct {
	// ArchiveName is the file name prefix and the top directory in the archive.
	ArchiveName string
	BuildInfo   component.BuildInfo
	Resource    map[string]string
	Logger      *zap.Logger
}

// Bundler serves support bundles over HTTP.
type Bundler struct {
	set       Settings
	available gatherers
	// busy stops concurrent bundles. Later gatherers change process-wide
	// profiling state, and each bundle can be large.
	busy atomic.Bool
}

// NewBundler returns a Bundler that can run the given gatherers.
func NewBundler(set Settings, entries []Entry) *Bundler {
	return &Bundler{set: set, available: newGatherers(entries)}
}

func (b *Bundler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !b.busy.CompareAndSwap(false, true) {
		http.Error(w, "a support bundle is already in progress", http.StatusTooManyRequests)
		return
	}
	defer b.busy.Store(false)

	requested := parseGatherers(r.URL.Query()[gatherersParam])
	start := time.Now()
	res, err := gather(r.Context(), b.available, requested)
	end := time.Now()
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, errNothingSelected) {
			status = http.StatusBadRequest
		}
		b.set.Logger.Warn("Support bundle not generated", zap.Error(err), zap.Any("skipped", res.selection.Skipped))
		http.Error(w, failureText(err, res.selection), status)
		return
	}

	m := newManifest(uuid.NewString(), start, end, b.set.BuildInfo, b.set.Resource, res)
	body, err := writeZip(b.set.ArchiveName, m, res.files)
	if err != nil {
		b.set.Logger.Error("Support bundle not generated", zap.Error(err))
		http.Error(w, "failed to write the support bundle", http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("%s-%s.zip", b.set.ArchiveName, m.Capture.Start.Format(filenameTimeFmt))
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	// The bundle can hold sensitive data, so no cache may keep a copy.
	w.Header().Set("Cache-Control", "no-store")
	if _, err := w.Write(body); err != nil {
		b.set.Logger.Warn("Failed to send the support bundle", zap.String("bundle_id", m.BundleID), zap.Error(err))
		return
	}
	b.set.Logger.Info("Support bundle generated",
		zap.String("bundle_id", m.BundleID),
		zap.Strings("ran", m.Gatherers.Ran),
		zap.Any("skipped", m.Gatherers.Skipped),
		zap.String("remote_addr", r.RemoteAddr))
}

// parseGatherers accepts comma-separated names in one or more query values.
// It removes empty names and duplicates, and keeps the request order.
func parseGatherers(values []string) []string {
	var names []string
	seen := map[string]struct{}{}
	for _, v := range values {
		for name := range strings.SplitSeq(v, ",") {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			names = append(names, name)
		}
	}
	return names
}

func failureText(err error, sel Selection) string {
	var sb strings.Builder
	sb.WriteString(err.Error())
	for _, s := range sel.Skipped {
		fmt.Fprintf(&sb, "\n%s: %s", s.Name, s.Reason)
		if s.Error != "" {
			fmt.Fprintf(&sb, ": %s", s.Error)
		}
	}
	return sb.String()
}

func writeZip(rootDir string, m Manifest, files []gatheredFile) ([]byte, error) {
	manifest, err := yaml.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("marshal manifest: %w", err)
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	write := func(name string, modified time.Time, content []byte) error {
		fw, err := zw.CreateHeader(&zip.FileHeader{
			Name:     path.Join(rootDir, name),
			Method:   zip.Deflate,
			Modified: modified,
		})
		if err != nil {
			return err
		}
		_, err = fw.Write(content)
		return err
	}

	if err := write(manifestPath, m.Capture.End, manifest); err != nil {
		return nil, err
	}
	for _, f := range files {
		if err := write(f.Path, f.collectedAt, f.Content); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
