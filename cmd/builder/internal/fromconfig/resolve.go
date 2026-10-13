// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package fromconfig // import "go.opentelemetry.io/collector/cmd/builder/internal/fromconfig"

import (
	"context"
	"fmt"
)

// Resolved is a component matched to a module.
type Resolved struct {
	Component
	// Module is the module path.
	Module string
	// Version is the version of the module, from the same release as the other resolved components.
	Version string
}

// GoMod returns the module specification in the format of the builder manifest.
func (r Resolved) GoMod() string {
	return r.Module + " " + r.Version
}

// Unresolved is a component that could not be matched to a module.
type Unresolved struct {
	Component
	// Reason describes why the component was not resolved.
	Reason string
}

// Result is the outcome of resolving the components of a configuration.
type Result struct {
	Resolved   []Resolved
	Unresolved []Unresolved
}

// Resolver resolves the components of a configuration to modules of a Collector release.
type Resolver struct {
	// Release is the Collector release the modules are selected from, such as v0.162.0.
	Release string
	// List looks up modules. Defaults to the Go toolchain.
	List ListFunc
	// Versions returns the versions of the release. Defaults to reading the go.mod of otelcol.
	Versions VersionsFunc
}

// Resolve matches the components referenced by the configuration to modules.
// Components that cannot be matched are reported in the result, not as an error.
func (r *Resolver) Resolve(ctx context.Context, config []byte) (Result, error) {
	components, err := ParseComponents(config)
	if err != nil {
		return Result{}, err
	}
	if len(components) == 0 {
		return Result{}, nil
	}

	list, versionsOf := r.List, r.Versions
	if list == nil {
		list = goList
	}
	if versionsOf == nil {
		versionsOf = goVersions
	}

	versions, err := versionsOf(ctx, r.Release)
	if err != nil {
		return Result{}, err
	}

	found, err := verify(ctx, list, components, func(m ListedModule) (string, bool) {
		return versions.selectVersion(m.Versions)
	})
	if err != nil {
		return Result{}, err
	}

	var result Result
	for _, c := range components {
		sel, ok := found.modules[c]
		if !ok {
			reason := fmt.Sprintf("no Go module found for %s %q", c.Class, c.Type)
			if found.found[c] {
				reason = fmt.Sprintf("the Go module of %s %q is not published at release %s", c.Class, c.Type, r.Release)
			}
			result.Unresolved = append(result.Unresolved, Unresolved{Component: c, Reason: reason})
			continue
		}
		result.Resolved = append(result.Resolved, Resolved{Component: c, Module: sel.module.Path, Version: sel.version})
	}
	return result, nil
}
