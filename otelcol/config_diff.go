// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol // import "go.opentelemetry.io/collector/otelcol"

import (
	"reflect"
	"slices"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/service/pipelines"
)

// ConfigChangeType categorizes the type of configuration change to determine
// the minimum reload scope required.
type ConfigChangeType int

const (
	// ConfigChangeNone indicates no configuration change.
	ConfigChangeNone ConfigChangeType = iota

	// ConfigChangePartialReload indicates receiver and/or processor
	// configurations changed, but exporters, connectors, extensions,
	// and pipeline structure are unchanged. A partial reload can handle this.
	ConfigChangePartialReload

	// ConfigChangeFullReload indicates changes to exporters, connectors,
	// extensions, telemetry, or pipeline structure that require a full reload.
	ConfigChangeFullReload
)

// categorizeConfigChange analyzes the differences between old and new configs
// and returns whether a partial reload can handle the change.
//
// The function checks configuration sections that require a full reload:
//  1. Service telemetry changes
//  2. Extension changes (config or list)
//
// Changes that can be handled by partial reload:
//   - Component config changes (receivers, processors, exporters, connectors)
//   - Adding/removing pipelines (with or without connectors)
//   - Adding/removing connectors as exporters in pipelines
//   - Adding/removing connectors as receivers in pipelines
//   - Adding/removing regular exporters in pipelines
//
// Graph.Reload handles the actual diff detection and only rebuilds affected
// pipelines.
//
// Configuration sections are compared using reflect.DeepEqual because
// component.Config is an empty interface with no hash or fingerprint contract.
func categorizeConfigChange(oldCfg, newCfg *Config, _ func(component.ID) bool) ConfigChangeType {
	// Service telemetry must be identical.
	if !reflect.DeepEqual(oldCfg.Service.Telemetry, newCfg.Service.Telemetry) {
		return ConfigChangeFullReload
	}

	// Extensions list must be identical.
	if !slices.Equal(oldCfg.Service.Extensions, newCfg.Service.Extensions) {
		return ConfigChangeFullReload
	}

	// Extension configs must be identical.
	if !reflect.DeepEqual(oldCfg.Extensions, newCfg.Extensions) {
		return ConfigChangeFullReload
	}

	// All other changes (pipelines, receivers, processors, exporters, connectors)
	// can be handled by partial reload. Graph.Reload will determine what
	// specific components need to be rebuilt.
	return ConfigChangePartialReload
}

// pipelineUsesConnectors returns true if the pipeline has any connectors
// as receivers or exporters.
func pipelineUsesConnectors(pipe *pipelines.PipelineConfig, isConnector func(component.ID) bool) bool {
	return slices.ContainsFunc(pipe.Receivers, isConnector) ||
		slices.ContainsFunc(pipe.Exporters, isConnector)
}

// filterIDs returns only the IDs for which the predicate returns true,
// preserving order.
func filterIDs(ids []component.ID, pred func(component.ID) bool) []component.ID {
	var out []component.ID
	for _, id := range ids {
		if pred(id) {
			out = append(out, id)
		}
	}
	return out
}

// isConnectorID returns a predicate function that checks whether a component.ID
// refers to a configured connector. This is used to distinguish connector-as-receiver
// entries from pure receivers in pipeline configs.
func isConnectorID(connectors map[component.ID]component.Config) func(component.ID) bool {
	return func(id component.ID) bool {
		_, ok := connectors[id]
		return ok
	}
}
