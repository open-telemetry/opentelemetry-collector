// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol // import "go.opentelemetry.io/collector/otelcol"

import (
	"reflect"
	"slices"

	"go.opentelemetry.io/collector/component"
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
// 1. Service telemetry, extensions
// 2. Pipeline structure changes (adding/removing pipelines, changing exporter
//    or connector-as-receiver lists within a pipeline)
//
// Changes to component configs (receivers, processors, exporters, connectors)
// can be handled by partial reload as long as the pipeline structure is unchanged.
// Graph.Reload handles the actual diff detection and only rebuilds affected
// pipelines.
//
// Configuration sections are compared using reflect.DeepEqual because
// component.Config is an empty interface with no hash or fingerprint contract.
//
// isConnector reports whether a given component.ID refers to a connector
// (as opposed to a regular receiver). Changes to connector-as-receiver
// entries in pipelines require a full reload.
func categorizeConfigChange(oldCfg, newCfg *Config, isConnector func(component.ID) bool) ConfigChangeType {
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

	// Must have the same set of pipeline IDs.
	if len(oldCfg.Service.Pipelines) != len(newCfg.Service.Pipelines) {
		return ConfigChangeFullReload
	}
	for pid := range oldCfg.Service.Pipelines {
		if _, ok := newCfg.Service.Pipelines[pid]; !ok {
			return ConfigChangeFullReload
		}
	}

	// Per-pipeline: exporter lists and connector-as-receiver entries must be identical.
	// Config changes to exporters/connectors are OK (handled by partial reload),
	// but structural changes (adding/removing from pipeline) require full reload.
	for pid, oldPipe := range oldCfg.Service.Pipelines {
		newPipe := newCfg.Service.Pipelines[pid]

		// Exporter list must be identical (can't add/remove exporters from pipeline).
		if !slices.Equal(oldPipe.Exporters, newPipe.Exporters) {
			return ConfigChangeFullReload
		}

		// Connector-as-receiver entries must match.
		oldConnReceivers := filterIDs(oldPipe.Receivers, isConnector)
		newConnReceivers := filterIDs(newPipe.Receivers, isConnector)
		if !slices.Equal(oldConnReceivers, newConnReceivers) {
			return ConfigChangeFullReload
		}
	}

	// Partial reload can handle component config changes.
	// Graph.Reload detects what changed and only rebuilds affected pipelines.
	return ConfigChangePartialReload
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
