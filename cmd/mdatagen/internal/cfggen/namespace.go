// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cfggen // import "go.opentelemetry.io/collector/cmd/mdatagen/internal/cfggen"

import (
	"errors"
	"fmt"
	"maps"
	"path"
	"regexp"
	"slices"
	"strings"
)

var namespaceToURL = map[string]string{
	"go.opentelemetry.io/collector":                             "https://raw.githubusercontent.com/open-telemetry/opentelemetry-collector",
	"github.com/open-telemetry/opentelemetry-collector-contrib": "https://raw.githubusercontent.com/open-telemetry/opentelemetry-collector-contrib",
}

var supportedNamespaces = slices.Collect(maps.Keys(namespaceToURL))

type RefKind int

const (
	External RefKind = iota
	Internal
	Local
)

type Ref struct {
	namespace string
	schemaID  string
	defName   string
	kind      RefKind
}

var localRefPattern = regexp.MustCompile(`^((?:/|\.\.?/).*?)(?:\.([^./]+))?$`)

func NewRef(refPath string) *Ref {
	var namespace, schemaID, defName string
	var kind RefKind

	switch {
	case localRefPattern.MatchString(refPath):
		matches := localRefPattern.FindStringSubmatch(refPath)
		schemaID = matches[1]
		defName = matches[2]
		kind = Local
	case !strings.ContainsRune(refPath, '/'):
		defName = refPath
		kind = Internal
	default:
		namespace = namespaceOf(refPath)
		rest, _ := strings.CutPrefix(refPath, namespace)
		schemaID, defName, _ = strings.Cut(rest, ".")
		schemaID = strings.Trim(schemaID, "/")
		kind = External
	}

	return &Ref{
		namespace,
		schemaID,
		defName,
		kind,
	}
}

func WithOrigin(refPath string, origin *Ref) *Ref {
	ref := NewRef(refPath)
	if origin != nil {
		if origin.isExternal() {
			ref.namespace = origin.namespace
			ref.kind = External
			if !strings.HasPrefix(ref.schemaID, "/") {
				ref.schemaID = path.Join(origin.schemaID, ref.schemaID)
			} else {
				ref.schemaID = strings.Trim(ref.schemaID, "/")
			}
		}
	}
	return ref
}

func namespaceOf(path string) string {
	if ns, ok := matchSupportedNamespace(path); ok {
		return ns
	}
	if idx := strings.LastIndex(path, "/"); idx != -1 {
		return path[:idx]
	}
	return ""
}

func matchSupportedNamespace(path string) (string, bool) {
	for _, ns := range supportedNamespaces {
		if strings.HasPrefix(path, ns) {
			return ns, true
		}
	}
	return "", false
}

func (r *Ref) Namespace() (string, bool) {
	_, ok := matchSupportedNamespace(r.namespace)
	return r.namespace, ok
}

func (r *Ref) Module() string {
	if r.namespace != "" {
		return r.namespace + "/" + r.schemaID
	}
	return ""
}

func (r *Ref) SchemaID() string {
	return r.schemaID
}

func (r *Ref) DefName() string {
	return r.defName
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
	return r.kind == Internal
}

func (r *Ref) isLocal() bool {
	return r.kind == Local
}

func (r *Ref) isExternal() bool {
	return r.kind == External
}

func (r *Ref) Validate() error {
	if r.String() == "" {
		return errors.New("empty path")
	}

	if r.defName == "" {
		return errors.New("missing definition name")
	}
	if r.isInternal() || r.isLocal() {
		if r.isLocal() && r.schemaID == "" {
			return errors.New("missing schema ID in local reference")
		}
	}

	return nil
}

func (r *Ref) String() string {
	var sb strings.Builder
	if r.namespace != "" {
		sb.WriteString(r.namespace)
	}
	if r.schemaID != "" {
		if sb.Len() > 0 {
			sb.WriteRune('/')
		}
		sb.WriteString(r.schemaID)
	}
	if r.defName != "" {
		if sb.Len() > 0 {
			sb.WriteRune('.')
		}
		sb.WriteString(r.defName)
	}

	return sb.String()
}

func (r *Ref) CacheKey() string {
	return r.String()
}
