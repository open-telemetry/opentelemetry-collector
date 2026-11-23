// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/mdatagen/internal"

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Discovery is the default discovery rule declared by a receiver or scraper.
type Discovery struct {
	// DefaultConfig is merged over the component's default config for every endpoint type.
	DefaultConfig map[string]any `mapstructure:"default_config"`
	// Endpoints holds the rule per endpoint type.
	Endpoints map[string]DiscoveryEndpoint `mapstructure:"endpoints"`
}

// DiscoveryEndpoint is the discovery rule for a single endpoint type.
type DiscoveryEndpoint struct {
	// Rule is an expression evaluated against the endpoint.
	Rule string `mapstructure:"rule"`
}

func (d *Discovery) validate(class string) error {
	if class != "receiver" && class != "scraper" {
		return fmt.Errorf("discovery is only supported for receivers and scrapers, got class %q", class)
	}
	if len(d.Endpoints) == 0 {
		return errors.New("discovery.endpoints must have at least one endpoint type")
	}
	var errs error
	for typ, e := range d.Endpoints {
		if strings.TrimSpace(e.Rule) == "" {
			errs = errors.Join(errs, fmt.Errorf("discovery.endpoints.%s.rule must be set", typ))
		}
	}
	return errs
}

// SortedEndpointTypes returns the endpoint types in alphabetical order.
func (d *Discovery) SortedEndpointTypes() []string {
	types := make([]string, 0, len(d.Endpoints))
	for typ := range d.Endpoints {
		types = append(types, typ)
	}
	slices.Sort(types)
	return types
}

// DefaultConfigLiteral returns DefaultConfig as a Go composite literal.
func (d *Discovery) DefaultConfigLiteral() string {
	if len(d.DefaultConfig) == 0 {
		return "map[string]any{}"
	}
	// fmt sorts map keys, so the output is deterministic.
	return strings.ReplaceAll(fmt.Sprintf("%#v", d.DefaultConfig), "interface {}", "any")
}
