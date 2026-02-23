// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cfggen // import "go.opentelemetry.io/collector/cmd/mdatagen/internal/cfggen"

import (
	"errors"
	"fmt"
	"maps"
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

func NewRef(path string) *Ref {
	return &Ref{path: path}
}

func (r *Ref) Namespace() (string, bool) {
	for _, namespace := range supportedNamespaces {
		if strings.HasPrefix(r.path, namespace) {
			return namespace, true
		}
	}
	return "", false
}

func (r *Ref) Module() string {
	pathAndVersion := strings.Split(r.path, "@")[0]
	moduleAndDef := strings.Split(pathAndVersion, ".")
	return strings.Join(moduleAndDef[:len(moduleAndDef)-1], ".")
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
