// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package supportbundle

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"go.yaml.in/yaml/v3"

	"go.opentelemetry.io/collector/component"
)

const archiveName = "test-bundle"

type fakeGatherer struct {
	name  string
	files []File
	err   error
	// started and release, when set, hold Gather until the test releases it.
	started chan struct{}
	release chan struct{}
}

func (g *fakeGatherer) Name() string { return g.name }

func (g *fakeGatherer) Gather(context.Context) ([]File, error) {
	if g.started != nil {
		close(g.started)
		<-g.release
	}
	return g.files, g.err
}

func newFake(name string, paths ...string) *fakeGatherer {
	g := &fakeGatherer{name: name}
	for _, p := range paths {
		g.files = append(g.files, File{Path: p, Content: []byte("content of " + p)})
	}
	return g
}

func newTestBundler(entries ...Entry) *Bundler {
	return NewBundler(Settings{
		ArchiveName: archiveName,
		BuildInfo:   component.BuildInfo{Command: "otelcol", Description: "test collector", Version: "1.2.3"},
		Resource:    map[string]string{"service.instance.id": "instance-1"},
		Logger:      zap.NewNop(),
	}, entries)
}

// bundle is a parsed support bundle.
type bundle struct {
	manifest Manifest
	files    map[string][]byte
}

func get(t *testing.T, b *Bundler, query string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	b.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/debug/v1/supportbundle"+query, http.NoBody))
	return rec
}

func readBundle(t *testing.T, body []byte) bundle {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	require.NoError(t, err)

	got := bundle{files: map[string][]byte{}}
	for _, f := range zr.File {
		rc, err := f.Open()
		require.NoError(t, err)
		content, err := io.ReadAll(rc)
		require.NoError(t, err)
		require.NoError(t, rc.Close())
		got.files[f.Name] = content
	}
	manifest, ok := got.files[archiveName+"/"+manifestPath]
	require.True(t, ok, "the bundle must hold a manifest")
	require.NoError(t, yaml.Unmarshal(manifest, &got.manifest))
	return got
}

func TestServeAllEnabled(t *testing.T) {
	b := newTestBundler(
		Entry{Gatherer: newFake("a", "a/one.txt", "a/two.txt"), Enabled: true},
		Entry{Gatherer: newFake("b", "b.txt"), Enabled: false},
		Entry{Gatherer: newFake("c", "c.txt"), Enabled: true},
	)

	rec := get(t, b, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "application/zip", rec.Header().Get("Content-Type"))
	assert.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
	assert.Contains(t, rec.Header().Get("Content-Disposition"), `filename="test-bundle-`)

	got := readBundle(t, rec.Body.Bytes())
	m := got.manifest
	assert.Equal(t, "v1", m.Format)
	assert.NotEmpty(t, m.BundleID)
	assert.False(t, m.Capture.End.Before(m.Capture.Start))
	assert.Equal(t, CollectorInfo{Command: "otelcol", Description: "test collector", Version: "1.2.3"}, m.Collector)
	assert.Equal(t, map[string]string{"service.instance.id": "instance-1"}, m.Resource)
	assert.NotEmpty(t, m.Runtime.GoVersion)

	assert.Empty(t, m.Gatherers.Requested)
	assert.Equal(t, []string{"a", "c"}, m.Gatherers.Ran)
	assert.Equal(t, []Skipped{{Name: "b", Reason: SkipDisabledInConfig}}, m.Gatherers.Skipped)

	// Every file is under the root directory and matches its manifest entry.
	require.Len(t, m.Files, 3)
	for _, fi := range m.Files {
		content, ok := got.files[archiveName+"/"+fi.Path]
		require.True(t, ok, fi.Path)
		sum := sha256.Sum256(content)
		assert.Equal(t, hex.EncodeToString(sum[:]), fi.SHA256, fi.Path)
		assert.Equal(t, len(content), fi.SizeBytes, fi.Path)
		assert.False(t, fi.CollectedAt.Before(m.Capture.Start), fi.Path)
	}
	assert.Equal(t, "a", m.Files[0].Gatherer)
	assert.Equal(t, "c", m.Files[2].Gatherer)
	assert.Len(t, got.files, 4)
}

func TestServeRequestedSelection(t *testing.T) {
	b := newTestBundler(
		Entry{Gatherer: newFake("a", "a.txt"), Enabled: true},
		Entry{Gatherer: newFake("b", "b.txt"), Enabled: false},
		Entry{Gatherer: newFake("c", "c.txt"), Enabled: true},
	)

	rec := get(t, b, "?gatherers=c,b,nope&gatherers=c")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	got := readBundle(t, rec.Body.Bytes())
	assert.Equal(t, Selection{
		Requested: []string{"c", "b", "nope"},
		Ran:       []string{"c"},
		Skipped: []Skipped{
			{Name: "b", Reason: SkipDisabledInConfig},
			{Name: "nope", Reason: SkipUnknown},
		},
	}, got.manifest.Gatherers)
	assert.Contains(t, got.files, archiveName+"/c.txt")
	assert.NotContains(t, got.files, archiveName+"/a.txt")
	assert.NotContains(t, got.files, archiveName+"/b.txt")
}

func TestServeOnlySkippedRequested(t *testing.T) {
	b := newTestBundler(
		Entry{Gatherer: newFake("a", "a.txt"), Enabled: true},
		Entry{Gatherer: newFake("b", "b.txt"), Enabled: false},
	)

	rec := get(t, b, "?gatherers=b,nope")
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "b: disabled_in_config")
	assert.Contains(t, rec.Body.String(), "nope: unknown")
}

func TestServeNothingEnabled(t *testing.T) {
	b := newTestBundler(Entry{Gatherer: newFake("a", "a.txt"), Enabled: false})

	rec := get(t, b, "")
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "a: disabled_in_config")
}

func TestServeGathererFailures(t *testing.T) {
	failing := newFake("failing", "failing.txt")
	failing.err = errors.New("boom")
	unavailable := &fakeGatherer{name: "unavailable", err: fmt.Errorf("hook is off: %w", ErrUnavailable)}

	b := newTestBundler(
		Entry{Gatherer: failing, Enabled: true},
		Entry{Gatherer: unavailable, Enabled: true},
		Entry{Gatherer: newFake("ok", "ok.txt"), Enabled: true},
	)

	rec := get(t, b, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	got := readBundle(t, rec.Body.Bytes())
	assert.Equal(t, []string{"ok"}, got.manifest.Gatherers.Ran)
	assert.Equal(t, []Skipped{
		{Name: "failing", Reason: SkipFailed, Error: "boom"},
		{Name: "unavailable", Reason: SkipUnavailable, Error: "hook is off: gatherer is unavailable"},
	}, got.manifest.Gatherers.Skipped)
	// A failed gatherer contributes no files, even if it returned some.
	assert.NotContains(t, got.files, archiveName+"/failing.txt")
}

func TestServeAllGatherersFail(t *testing.T) {
	failing := &fakeGatherer{name: "failing", err: errors.New("boom")}
	b := newTestBundler(Entry{Gatherer: failing, Enabled: true})

	rec := get(t, b, "")
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "failing: failed: boom")
}

func TestServeInvalidPaths(t *testing.T) {
	tests := []struct {
		name  string
		paths []string
		want  string
	}{
		{name: "parent", paths: []string{"../escape.txt"}, want: `invalid file path "../escape.txt"`},
		{name: "only parent", paths: []string{".."}, want: `invalid file path ".."`},
		{name: "inner parent", paths: []string{"a/../../escape.txt"}, want: `invalid file path "a/../../escape.txt"`},
		{name: "root", paths: []string{"."}, want: `invalid file path "."`},
		{name: "empty", paths: []string{""}, want: `invalid file path ""`},
		{name: "backslash parent", paths: []string{`..\escape.txt`}, want: `invalid file path "..\\escape.txt"`},
		{name: "absolute", paths: []string{"/etc/passwd"}, want: `invalid file path "/etc/passwd"`},
		{name: "unclean", paths: []string{"a//b.txt"}, want: `invalid file path "a//b.txt"`},
		{name: "manifest", paths: []string{manifestPath}, want: `file path "manifest.yaml" is already used by "bundle"`},
		{name: "other gatherer", paths: []string{"first.txt"}, want: `file path "first.txt" is already used by "first"`},
		{name: "twice", paths: []string{"x.txt", "x.txt"}, want: `file path "x.txt" is used more than once`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := newTestBundler(
				Entry{Gatherer: newFake("first", "first.txt"), Enabled: true},
				Entry{Gatherer: newFake("bad", tt.paths...), Enabled: true},
			)
			rec := get(t, b, "")
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

			got := readBundle(t, rec.Body.Bytes())
			assert.Equal(t, []string{"first"}, got.manifest.Gatherers.Ran)
			assert.Equal(t, []Skipped{{Name: "bad", Reason: SkipFailed, Error: tt.want}}, got.manifest.Gatherers.Skipped)
		})
	}
}

func TestServeRejectsConcurrentBundles(t *testing.T) {
	slow := newFake("slow", "slow.txt")
	slow.started = make(chan struct{})
	slow.release = make(chan struct{})
	b := newTestBundler(Entry{Gatherer: slow, Enabled: true})

	done := make(chan *httptest.ResponseRecorder)
	go func() { done <- get(t, b, "") }()
	<-slow.started

	assert.Equal(t, http.StatusTooManyRequests, get(t, b, "").Code)

	close(slow.release)
	assert.Equal(t, http.StatusOK, (<-done).Code)

	// The bundler accepts a new request when the first one is done.
	slow.started, slow.release = nil, nil
	assert.Equal(t, http.StatusOK, get(t, b, "").Code)
}

func TestServeLogsSkippedWithManifestKeys(t *testing.T) {
	var logs bytes.Buffer
	core := zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(&logs), zap.InfoLevel)
	b := NewBundler(Settings{ArchiveName: archiveName, Logger: zap.New(core)},
		[]Entry{{Gatherer: newFake("a", "a.txt"), Enabled: true}})

	require.Equal(t, http.StatusOK, get(t, b, "?gatherers=a,nope").Code)
	assert.Contains(t, logs.String(), `"skipped":[{"name":"nope","reason":"unknown"}]`)
}

// failingResponseWriter accepts headers but fails every body write.
type failingResponseWriter struct {
	*httptest.ResponseRecorder
}

func (failingResponseWriter) Write([]byte) (int, error) {
	return 0, errors.New("connection reset")
}

func TestServeResponseWriteError(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	b := NewBundler(Settings{ArchiveName: archiveName, Logger: zap.New(core)},
		[]Entry{{Gatherer: newFake("a", "a.txt"), Enabled: true}})

	b.ServeHTTP(failingResponseWriter{httptest.NewRecorder()},
		httptest.NewRequest(http.MethodGet, "/debug/v1/supportbundle", http.NoBody))

	require.Equal(t, 1, logs.Len())
	entry := logs.All()[0]
	assert.Equal(t, "Failed to send the support bundle", entry.Message)
	assert.Equal(t, "connection reset", entry.ContextMap()["error"])
}

// limitWriter accepts n bytes and then fails.
type limitWriter struct {
	n int
}

func (w *limitWriter) Write(p []byte) (int, error) {
	if len(p) > w.n {
		written := w.n
		w.n = 0
		return written, errors.New("write limit reached")
	}
	w.n -= len(p)
	return len(p), nil
}

func TestWriteZipWriteErrors(t *testing.T) {
	// Random content does not compress, so the archive holds the full size and
	// every write reaches the output.
	rnd := rand.New(rand.NewPCG(1, 2))
	newContent := func() []byte {
		content := make([]byte, 32*1024)
		for i := range content {
			content[i] = byte(rnd.Uint32())
		}
		return content
	}
	files := []gatheredFile{
		{File: File{Path: "a.bin", Content: newContent()}, gatherer: "a"},
		{File: File{Path: "b.bin", Content: newContent()}, gatherer: "b"},
	}
	m := Manifest{Format: formatVersion, BundleID: "id"}

	var full bytes.Buffer
	require.NoError(t, writeZip(&full, archiveName, m, files))

	// The archive must fail at every point where the output stops accepting data.
	for limit := 0; limit < full.Len(); limit += full.Len()/64 + 1 {
		require.Error(t, writeZip(&limitWriter{n: limit}, archiveName, m, files), "limit %d", limit)
	}
	assert.NoError(t, writeZip(&limitWriter{n: full.Len()}, archiveName, m, files))
}

func TestServeMethodNotAllowed(t *testing.T) {
	b := newTestBundler(Entry{Gatherer: newFake("a", "a.txt"), Enabled: true})

	rec := httptest.NewRecorder()
	b.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/debug/v1/supportbundle", http.NoBody))
	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
	assert.Equal(t, http.MethodGet, rec.Header().Get("Allow"))
}

func TestParseGatherers(t *testing.T) {
	assert.Nil(t, parseGatherers(nil))
	assert.Nil(t, parseGatherers([]string{"", " , "}))
	assert.Equal(t, []string{"a", "b", "c"}, parseGatherers([]string{" a,b ", "a", "c,,b"}))
}
