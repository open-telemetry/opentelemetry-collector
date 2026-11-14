// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package xcomponentdiscovery defines the default discovery rule that a
// component can declare, so that a consumer such as receiver_creator can start
// the component for matching endpoints reported by observers.
//
// This package only holds declarations. Evaluating rules is up to the consumer.
// This package is experimental and can change without a deprecation cycle.
package xcomponentdiscovery // import "go.opentelemetry.io/collector/component/xcomponentdiscovery"

// EndpointType is the type of endpoint reported by an observer,
// e.g. "port", "hostport", "container". Values are defined by observers.
type EndpointType string

// Rule describes how endpoints for a component are discovered and the
// configuration the component is started with.
type Rule struct {
	// DefaultConfig is merged over the component's default config for every
	// endpoint type. It can use the expressions supported by the consumer,
	// e.g. `endpoint` in receiver_creator.
	DefaultConfig map[string]any
	// Endpoints holds the rule per endpoint type. Endpoint types without an
	// entry are not matched.
	Endpoints map[EndpointType]EndpointRule
}

// EndpointRule is the discovery rule for a single endpoint type.
type EndpointRule struct {
	// Rule is an expression evaluated against the endpoint by the consumer,
	// e.g. receiver_creator. The endpoint type check is implied by the map
	// key and must not be part of the rule.
	Rule string
}

// IsEmpty returns true if the rule doesn't match any endpoint type.
func (r Rule) IsEmpty() bool {
	return len(r.Endpoints) == 0
}
