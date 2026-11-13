// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package fromconfig // import "go.opentelemetry.io/collector/cmd/builder/internal/fromconfig"

import (
	"slices"
	"strings"

	"golang.org/x/mod/module"
)

// The module path prefixes of the repositories hosting official components.
const (
	CorePrefix    = "go.opentelemetry.io/collector/"
	ContribPrefix = "github.com/open-telemetry/opentelemetry-collector-contrib/"
)

// prefixes lists the repositories to search, in priority order.
var prefixes = []string{CorePrefix, ContribPrefix}

// exceptions lists the components whose module is not reachable by any pattern.
var exceptions = map[Component]string{
	{Class: Extension, Type: "asapclient"}:  ContribPrefix + "extension/asapauthextension",
	{Class: Receiver, Type: "podman_stats"}: ContribPrefix + "receiver/podmanreceiver",
	{Class: Exporter, Type: "otlp_grpc"}:    CorePrefix + "exporter/otlpexporter",
}

// candidate is a module path that may host a component.
type candidate struct {
	// level is the priority of the candidate; lower is tried first.
	level int
	path  string
}

// candidates returns the module paths that may host the component, in
// priority order. Paths that are not valid module paths are omitted.
func candidates(c Component) []candidate {
	if c.Type == "" {
		return nil
	}
	if path, ok := exceptions[c]; ok {
		return []candidate{{level: 0, path: path}}
	}

	var result []candidate
	for i, suffix := range suffixes(c) {
		for j, prefix := range prefixes {
			path := prefix + suffix
			if module.CheckPath(path) != nil || isNotComponent(c.Class, path) {
				continue
			}
			if slices.ContainsFunc(result, func(o candidate) bool { return o.path == path }) {
				continue
			}
			result = append(result, candidate{level: i*len(prefixes) + j, path: path})
		}
	}
	return result
}

// suffixes returns the module path suffixes for the component, in priority order.
func suffixes(c Component) []string {
	class := string(c.Class)
	segs := strings.Split(c.Type, "_")
	t := strings.Join(segs, "")

	out := []string{
		class + "/" + t + class,
		class + "/" + t,
	}
	if c.Class == Extension {
		out = append(out, class+"/"+t+"auth"+class)
	}
	if len(segs) > 1 {
		last := segs[len(segs)-1]
		rest := strings.Join(segs[:len(segs)-1], "")
		reversed := slices.Clone(segs)
		slices.Reverse(reversed)
		out = append(out,
			class+"/"+last+"/"+rest+last+class,
			class+"/"+last+"/"+rest+last,
			class+"/"+strings.Join(reversed, "")+class,
		)
	}
	return out
}
