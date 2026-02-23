// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cfggen // import "go.opentelemetry.io/collector/cmd/mdatagen/internal/cfggen"

import (
	"errors"
	"fmt"
	"maps"
	"path"
	"slices"
	"strings"
)

var namespaceToURL = map[string]string{
	"go.opentelemetry.io/collector":                             "https://raw.githubusercontent.com/open-telemetry/opentelemetry-collector",
	"github.com/open-telemetry/opentelemetry-collector-contrib": "https://raw.githubusercontent.com/open-telemetry/opentelemetry-collector-contrib",
}

var supportedNamespaces = slices.Collect(maps.Keys(namespaceToURL))

type Ref struct {
	path string
}

// NewRef creates a new Ref from a path string. If origin is non-empty (the module
// path of the schema containing the ref, e.g. "go.opentelemetry.io/collector/config/confighttp"),
// local refs are converted to fully qualified external refs:
//   - "/config/configauth.config"       → namespace + path (absolute within namespace)
//   - "./internal/metadata.config"      → origin + "/internal/metadata.config" (relative to schema)
//   - "../configtls.config"             → parent(origin) + "/configtls.config" (parent-relative)
func NewRef(refPath, origin string) *Ref {
	if origin != "" {
		// Split off inline version from origin (e.g. "mod/path@v1.2.0" → "mod/path", "v1.2.0")
		originModule, originVersion, _ := strings.Cut(origin, "@")

		var resolved bool
		switch {
		case strings.HasPrefix(refPath, "/"):
			if ns := namespaceOf(originModule); ns != "" {
				refPath = path.Join(ns, refPath)
				resolved = true
			}
		case strings.HasPrefix(refPath, "./"), strings.HasPrefix(refPath, "../"):
			refPath = path.Join(originModule, refPath)
			resolved = true
		}

		if resolved && originVersion != "" {
			refPath += "@" + originVersion
		}
	}
	return &Ref{path: refPath}
}

func namespaceOf(path string) string {
	for _, namespace := range supportedNamespaces {
		if strings.HasPrefix(path, namespace) {
			return namespace
		}
	}
	return ""
}

func (r *Ref) Namespace() (string, bool) {
	ns := namespaceOf(r.path)
	return ns, ns != ""
}

func (r *Ref) Module() string {
	pathAndVersion := strings.Split(r.path, "@")[0]
	moduleAndDef := strings.Split(pathAndVersion, ".")
	return strings.Join(moduleAndDef[:len(moduleAndDef)-1], ".")
}

// Origin returns the module path with inline version (if any), suitable for
// passing as origin to NewRef when resolving local refs inside this schema.
func (r *Ref) Origin() string {
	origin := r.Module()
	if v := r.InlineVersion(); v != "" {
		origin += "@" + v
	}
	return origin
}

func (r *Ref) SchemaID() string {
	module := r.Module()
	namespace, ok := r.Namespace()
	if ok {
		return strings.TrimPrefix(module, namespace+"/")
	}
	return module
}

func (r *Ref) DefName() string {
	if r.isInternal() {
		return r.path
	}
	pathAndVersion := strings.Split(r.path, "@")
	schemaAndDef := strings.Split(pathAndVersion[0], ".")
	return schemaAndDef[len(schemaAndDef)-1]
}

func (r *Ref) InlineVersion() string {
	pathAndVersion := strings.Split(r.path, "@")
	if len(pathAndVersion) == 2 {
		return pathAndVersion[1]
	}
	return ""
}

func (r *Ref) URL(version string) (string, error) {
	ns, ok := r.Namespace()
	if !ok {
		return "", errors.New("unsupported namespace")
	}
	baseURL := namespaceToURL[ns]
	return fmt.Sprintf("%s/%s/%s/%s",
			baseURL,
			version,
			r.SchemaID(),
			schemaFileName),
		nil
}

func (r *Ref) isInternal() bool {
	return !strings.ContainsRune(r.path, '/')
}

func (r *Ref) isLocal() bool {
	return strings.HasPrefix(r.path, "./") || strings.HasPrefix(r.path, "../") || strings.HasPrefix(r.path, "/")
}

func (r *Ref) isExternal() bool {
	return !r.isInternal() && !r.isLocal()
}

func (r *Ref) Validate() error {
	if r.path == "" {
		return errors.New("empty path")
	}
	inlineVersion := r.InlineVersion()
	if inlineVersion != "" {
		if r.isInternal() || r.isLocal() {
			return errors.New("inline version is not allowed for internal or local references")
		}
	}

	if r.isInternal() && strings.Contains(r.path, ".") {
		return errors.New("internal references should not contain '.' character")
	}

	if r.DefName() == "" {
		return errors.New("reference must contain a definition part")
	}

	return nil
}

func (r *Ref) CacheKey() string {
	return r.path
}

func (r *Ref) String() string {
	return r.path
}
