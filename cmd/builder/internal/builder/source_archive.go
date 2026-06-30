// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package builder // import "go.opentelemetry.io/collector/cmd/builder/internal/builder"

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
	"golang.org/x/mod/modfile"
)

const (
	// completeMarker marks a cache directory as fully extracted and reusable.
	completeMarker = ".complete"

	// maxDecompressedSize and maxFileCount bound decompression bombs.
	maxDecompressedSize = 8 << 30
	maxFileCount        = 1 << 20

	httpClientTimeout = 5 * time.Minute
)

// resolveSourceArchives downloads, verifies and extracts every module's source archive.
func (c *Config) resolveSourceArchives() error {
	lists := [][]Module{
		c.Extensions,
		c.Receivers,
		c.Exporters,
		c.Processors,
		c.Connectors,
		{c.Telemetry},
		c.ConfmapProviders,
		c.ConfmapConverters,
	}
	for _, list := range lists {
		for i := range list {
			if list[i].SourceArchive == nil {
				continue
			}
			if err := c.resolveSourceArchive(&list[i]); err != nil {
				return err
			}
		}
	}
	// c.Telemetry was passed by a one-element slice literal above, so copy back.
	c.Telemetry = lists[5][0]
	return nil
}

// sourceArchiveVersion is the synthetic require version for archive modules. It
// is never resolved (a directory replace bypasses the proxy) but must be valid semver.
const sourceArchiveVersion = "v0.0.0-sourcearchive"

func (c *Config) resolveSourceArchive(mod *Module) error {
	sa := mod.SourceArchive

	wantHash, err := c.resolveExpectedHash(sa)
	if err != nil {
		return fmt.Errorf("source_archive %q: %w", sa.URL, err)
	}

	cacheRoot, err := c.sourceArchiveCache()
	if err != nil {
		return err
	}
	cacheDir := filepath.Join(cacheRoot, wantHash)

	effectiveRoot, modulePath, err := c.ensureExtracted(cacheRoot, cacheDir, sa, wantHash)
	if err != nil {
		return fmt.Errorf("source_archive %q: %w", sa.URL, err)
	}

	// Runs on cache hits too: mod.Import can differ between runs.
	if err := validateArchiveImport(mod.Import, modulePath, effectiveRoot); err != nil {
		return fmt.Errorf("source_archive %q: %w", sa.URL, err)
	}

	c.Logger.Info("Resolved source archive", zap.String("module", modulePath), zap.String("import", mod.Import), zap.String("path", effectiveRoot))
	mod.Path = effectiveRoot
	mod.GoMod = modulePath + " " + sourceArchiveVersion
	mod.fromSourceArchive = true
	return nil
}

// validateArchiveImport verifies mod.Import is within the archive's module and
// not shadowed by a nested module along the import path.
func validateArchiveImport(importPath, modulePath, effectiveRoot string) error {
	if importPath != modulePath && !strings.HasPrefix(importPath, modulePath+"/") {
		return fmt.Errorf("import %q is not within module %q provided by the archive — the component would not be built from the archive", importPath, modulePath)
	}

	// A go.mod along the import path is a nested module that would own the import,
	// which the outer module's replace would not cover.
	rel := strings.TrimPrefix(importPath, modulePath)
	rel = strings.TrimPrefix(rel, "/")
	if rel == "" {
		return nil
	}
	segments := strings.Split(rel, "/")
	dir := effectiveRoot
	for _, seg := range segments {
		dir = filepath.Join(dir, seg)
		if fileExists(filepath.Join(dir, "go.mod")) {
			nested := modulePath + "/" + strings.TrimPrefix(strings.TrimPrefix(dir, effectiveRoot), string(filepath.Separator))
			nested = filepath.ToSlash(nested)
			return fmt.Errorf("import %q is shadowed by nested module %q in the archive — its replace would not cover the import", importPath, nested)
		}
	}
	return nil
}

// sourceArchiveCache returns the root directory used to cache source archives.
func (c *Config) sourceArchiveCache() (string, error) {
	if c.sourceArchiveCacheRoot != "" {
		return c.sourceArchiveCacheRoot, nil
	}
	if c.DownloadCacheDir != "" {
		return c.DownloadCacheDir, nil
	}
	userCache, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine user cache dir: %w", err)
	}
	return filepath.Join(userCache, "otelcol-builder", "source_archive"), nil
}

func (c *Config) resolveExpectedHash(sa *SourceArchive) (string, error) {
	if sa.SHA256 != "" {
		return strings.ToLower(sa.SHA256), nil
	}

	body, err := c.fetchBytes(sa.SHA256URL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch sha256_url: %w", err)
	}

	base := path.Base(mustURLPath(sa.URL))
	for line := range strings.SplitSeq(string(body), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		// sha256sum format: "<hex>  <filename>", '*' prefix for binary mode.
		name := strings.TrimPrefix(fields[len(fields)-1], "*")
		if path.Base(name) == base {
			return strings.ToLower(fields[0]), nil
		}
	}
	return "", fmt.Errorf("sha256_url %q does not contain an entry for %q", sa.SHA256URL, base)
}

// ensureExtracted populates cacheDir with the verified, extracted archive and
// returns the effective module root and the module path from its go.mod. The
// download is skipped when cacheDir already has a completion marker.
func (c *Config) ensureExtracted(cacheRoot, cacheDir string, sa *SourceArchive, wantHash string) (string, string, error) {
	if marker := filepath.Join(cacheDir, completeMarker); fileExists(marker) {
		c.Logger.Info("Using cached source archive", zap.String("path", cacheDir))
		return c.effectiveRoot(cacheDir, sa)
	}

	if err := os.MkdirAll(cacheRoot, 0o750); err != nil {
		return "", "", fmt.Errorf("failed to create cache root: %w", err)
	}

	archivePath, err := c.downloadArchive(cacheRoot, sa, wantHash)
	if err != nil {
		return "", "", err
	}
	defer os.Remove(archivePath)

	tmpDir, err := os.MkdirTemp(cacheRoot, "extract-")
	if err != nil {
		return "", "", fmt.Errorf("failed to create temp extraction dir: %w", err)
	}
	// On any error, clean up the partial extraction.
	committed := false
	defer func() {
		if !committed {
			os.RemoveAll(tmpDir)
		}
	}()

	if err := extractArchive(archivePath, sa.URL, tmpDir); err != nil {
		return "", "", err
	}

	if _, _, err := c.effectiveRoot(tmpDir, sa); err != nil {
		return "", "", err
	}

	if err := os.WriteFile(filepath.Join(tmpDir, completeMarker), nil, 0o600); err != nil {
		return "", "", fmt.Errorf("failed to write completion marker: %w", err)
	}
	if err := os.Rename(tmpDir, cacheDir); err != nil {
		// Lost-race case: another process already populated the cache dir.
		if fileExists(filepath.Join(cacheDir, completeMarker)) {
			os.RemoveAll(tmpDir)
			committed = true
			return c.effectiveRoot(cacheDir, sa)
		}
		return "", "", fmt.Errorf("failed to move extracted archive into cache: %w", err)
	}
	committed = true

	return c.effectiveRoot(cacheDir, sa)
}

// effectiveRoot resolves the module root within an extracted tree (auto-stripping
// a lone top-level directory and applying subdir) and returns it with the module
// path read from the resolved go.mod.
func (c *Config) effectiveRoot(root string, sa *SourceArchive) (string, string, error) {
	effective := root

	// Auto-strip a lone top-level directory when there is no top-level go.mod.
	if !fileExists(filepath.Join(effective, "go.mod")) {
		entries, err := os.ReadDir(effective)
		if err != nil {
			return "", "", fmt.Errorf("failed to read extracted root: %w", err)
		}
		var dirs []string
		extraFiles := false
		for _, e := range entries {
			switch {
			case e.IsDir():
				dirs = append(dirs, e.Name())
			case e.Name() == completeMarker:
				// ignore the completion marker written by ensureExtracted
			default:
				extraFiles = true
			}
		}
		if len(dirs) == 1 && !extraFiles {
			effective = filepath.Join(effective, dirs[0])
		}
	}

	if sa.Subdir != "" {
		cleaned := filepath.Clean(sa.Subdir)
		if filepath.IsAbs(cleaned) || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
			return "", "", fmt.Errorf("subdir %q escapes the archive tree", sa.Subdir)
		}
		candidate := filepath.Join(effective, cleaned)
		rel, err := filepath.Rel(effective, candidate)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return "", "", fmt.Errorf("subdir %q escapes the archive tree", sa.Subdir)
		}
		effective = candidate
	}

	goModPath := filepath.Join(effective, "go.mod")
	data, err := os.ReadFile(filepath.Clean(goModPath))
	if err != nil {
		return "", "", fmt.Errorf("expected go.mod at %q: %w", goModPath, err)
	}
	parsed, err := modfile.Parse(goModPath, data, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse go.mod at %q: %w", goModPath, err)
	}
	if parsed.Module == nil {
		return "", "", fmt.Errorf("go.mod at %q has no module path", goModPath)
	}

	return effective, parsed.Module.Mod.Path, nil
}

// downloadArchive downloads and sha256-verifies the archive, retrying on failure.
func (c *Config) downloadArchive(cacheRoot string, sa *SourceArchive, wantHash string) (string, error) {
	var failReason string
	for i := 1; i <= c.downloadModules.numRetries; i++ {
		archivePath, err := c.downloadArchiveOnce(cacheRoot, sa, wantHash)
		if err == nil {
			return archivePath, nil
		}
		failReason = err.Error()
		c.Logger.Info("Failed source archive download", zap.String("retry", fmt.Sprintf("%d/%d", i, c.downloadModules.numRetries)), zap.String("error", failReason))
		if i < c.downloadModules.numRetries {
			time.Sleep(c.downloadModules.wait)
		}
	}
	return "", fmt.Errorf("failed to download source archive: %s", failReason)
}

func (c *Config) downloadArchiveOnce(cacheRoot string, sa *SourceArchive, wantHash string) (string, error) {
	c.Logger.Info("Downloading source archive", zap.String("url", sa.URL))

	rc, err := c.openURL(sa.URL)
	if err != nil {
		return "", err
	}
	defer rc.Close()

	tmp, err := os.CreateTemp(cacheRoot, "download-")
	if err != nil {
		return "", fmt.Errorf("failed to create temp download file: %w", err)
	}
	tmpName := tmp.Name()

	hasher := sha256.New()
	if _, err = io.Copy(tmp, io.TeeReader(rc, hasher)); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return "", fmt.Errorf("failed to read source archive: %w", err)
	}
	if err = tmp.Close(); err != nil {
		os.Remove(tmpName)
		return "", fmt.Errorf("failed to close download file: %w", err)
	}

	gotHash := hex.EncodeToString(hasher.Sum(nil))
	if gotHash != wantHash {
		os.Remove(tmpName)
		return "", fmt.Errorf("sha256 mismatch: got %s, want %s", gotHash, wantHash)
	}
	return tmpName, nil
}

// fileURLPath converts a file:// URL to an OS path, handling Windows drive paths
// (file:///C:/dir -> C:\dir).
func fileURLPath(u *url.URL) string {
	p := u.Path
	if len(p) >= 3 && p[0] == '/' && p[2] == ':' {
		p = p[1:]
	}
	return filepath.FromSlash(p)
}

// openURL opens an https or file URL for reading.
func (c *Config) openURL(raw string) (io.ReadCloser, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid URL %q: %w", raw, err)
	}
	switch {
	case u.Scheme == "file":
		f, err := os.Open(filepath.Clean(fileURLPath(u)))
		if err != nil {
			return nil, err
		}
		return f, nil
	case u.Scheme == "https", u.Scheme == "http" && c.httpClient != nil:
		// http is honored only with an injected client (test-only).
		client := c.httpClient
		if client == nil {
			client = &http.Client{Timeout: httpClientTimeout}
		}
		//nolint:noctx // download uses a client-level timeout
		resp, err := client.Get(raw)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("unexpected status %s for %q", resp.Status, raw)
		}
		return resp.Body, nil
	default:
		return nil, fmt.Errorf("unsupported URL scheme %q", u.Scheme)
	}
}

// fetchBytes reads the full contents of an https or file URL.
func (c *Config) fetchBytes(raw string) ([]byte, error) {
	rc, err := c.openURL(raw)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(io.LimitReader(rc, 1<<20))
}

// extractArchive extracts a .tar.gz/.tgz or .zip archive into dst.
func extractArchive(archivePath, sourceURL, dst string) error {
	switch {
	case hasSuffixFold(sourceURL, ".tar.gz"), hasSuffixFold(sourceURL, ".tgz"):
		return extractTarGz(archivePath, dst)
	case hasSuffixFold(sourceURL, ".zip"):
		return extractZip(archivePath, dst)
	default:
		return fmt.Errorf("unsupported archive type for %q (expected .tar.gz, .tgz or .zip)", sourceURL)
	}
}

func extractTarGz(archivePath, dst string) error {
	f, err := os.Open(filepath.Clean(archivePath))
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("failed to open gzip stream: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	var total int64
	var count int
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry: %w", err)
		}

		count++
		if count > maxFileCount {
			return fmt.Errorf("archive exceeds maximum file count of %d", maxFileCount)
		}

		target, err := sanitizeEntryPath(dst, hdr.Name)
		if err != nil {
			return err
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o750); err != nil {
				return err
			}
		case tar.TypeReg:
			written, err := writeRegularFile(target, tr, hdr.FileInfo().Mode(), total)
			if err != nil {
				return err
			}
			total = written
		default:
			// Skip symlinks, devices, etc.
			continue
		}
	}
	return nil
}

func extractZip(archivePath, dst string) error {
	zr, err := zip.OpenReader(filepath.Clean(archivePath))
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}
	defer zr.Close()

	var total int64
	var count int
	for _, file := range zr.File {
		count++
		if count > maxFileCount {
			return fmt.Errorf("archive exceeds maximum file count of %d", maxFileCount)
		}

		target, err := sanitizeEntryPath(dst, file.Name)
		if err != nil {
			return err
		}

		if file.FileInfo().IsDir() {
			if mkErr := os.MkdirAll(target, 0o750); mkErr != nil {
				return mkErr
			}
			continue
		}
		if !file.Mode().IsRegular() {
			// Skip symlinks, devices, etc.
			continue
		}

		rc, err := file.Open()
		if err != nil {
			return err
		}
		written, err := writeRegularFile(target, rc, file.Mode(), total)
		rc.Close()
		if err != nil {
			return err
		}
		total = written
	}
	return nil
}

// sanitizeEntryPath validates an archive entry name and returns the absolute
// destination path, rejecting absolute paths and parent-directory traversal.
func sanitizeEntryPath(dst, name string) (string, error) {
	slashed := filepath.ToSlash(name)
	if name == "" || slashed == "/" {
		return "", fmt.Errorf("archive entry %q has an empty name", name)
	}
	if filepath.IsAbs(name) || strings.HasPrefix(slashed, "/") {
		return "", fmt.Errorf("archive entry %q has an absolute path", name)
	}
	// Clean unrooted so traversal segments survive for detection below.
	clean := filepath.Clean(filepath.FromSlash(name))
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("archive entry %q escapes the destination directory", name)
	}
	target := filepath.Join(dst, clean)
	rel, err := filepath.Rel(dst, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("archive entry %q escapes the destination directory", name)
	}
	return target, nil
}

// writeRegularFile writes the contents of r to target, enforcing the total
// decompressed-size cap, and preserving only the exec bit of the mode.
func writeRegularFile(target string, r io.Reader, mode os.FileMode, total int64) (int64, error) {
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return total, err
	}

	perm := os.FileMode(0o644)
	if mode&0o100 != 0 {
		perm = 0o755
	}

	out, err := os.OpenFile(filepath.Clean(target), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return total, err
	}
	defer out.Close()

	remaining := max(maxDecompressedSize-total, 0)
	// Allow one extra byte so we can detect overflow.
	n, err := io.Copy(out, io.LimitReader(r, remaining+1))
	total += n
	if err != nil {
		return total, err
	}
	if total > maxDecompressedSize {
		return total, fmt.Errorf("archive exceeds maximum decompressed size of %d bytes", maxDecompressedSize)
	}
	return total, nil
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func hasSuffixFold(s, suffix string) bool {
	return len(s) >= len(suffix) && strings.EqualFold(s[len(s)-len(suffix):], suffix)
}

func mustURLPath(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	return u.Path
}
