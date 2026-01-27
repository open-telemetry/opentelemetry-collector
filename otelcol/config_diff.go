// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol // import "go.opentelemetry.io/collector/otelcol"

import (
	"reflect"
	"slices"

	"go.opentelemetry.io/collector/component"
)

// receiversOnlyChange returns true when the only differences between old and
// new are in receiver configurations and/or the set of pure-receiver entries
// in pipeline receiver lists. Everything else (processors, exporters,
// connectors, extensions, telemetry, pipeline structure) must be identical.
//
// Configuration sections are compared using reflect.DeepEqual because
// component.Config is an empty interface with no hash or fingerprint contract.
// Serialization-based comparison (JSON, gob, etc.) is unsafe here because
// configopaque.String implements MarshalText by returning "[REDACTED]",
// which would treat configs with different secret values as identical and
// silently skip necessary reloads. reflect.DeepEqual compares raw field
// values without invoking marshal interfaces, so it correctly distinguishes
// configs that differ only in opaque fields. A false negative (reporting
// equal configs as different) is safe — it simply falls back to a full reload.
//
// isConnector reports whether a given component.ID refers to a connector
// (as opposed to a regular receiver). Changes to connector-as-receiver
// entries require a full reload.
func receiversOnlyChange(old, new *Config, isConnector func(component.ID) bool) bool {
	// Service telemetry must be identical.
	if !reflect.DeepEqual(old.Service.Telemetry, new.Service.Telemetry) {
		return false
	}

	// Extensions list must be identical.
	if !slices.Equal(old.Service.Extensions, new.Service.Extensions) {
		return false
	}

	// Extension configs must be identical.
	if !reflect.DeepEqual(old.Extensions, new.Extensions) {
		return false
	}

	// Processor configs must be identical.
	if !reflect.DeepEqual(old.Processors, new.Processors) {
		return false
	}

	// Exporter configs must be identical.
	if !reflect.DeepEqual(old.Exporters, new.Exporters) {
		return false
	}

	// Connector configs must be identical.
	if !reflect.DeepEqual(old.Connectors, new.Connectors) {
		return false
	}

	// Must have the same set of pipeline IDs.
	if len(old.Service.Pipelines) != len(new.Service.Pipelines) {
		return false
	}
	for pid := range old.Service.Pipelines {
		if _, ok := new.Service.Pipelines[pid]; !ok {
			return false
		}
	}

	// Per-pipeline: processors, exporters, and connector-as-receiver entries
	// must be identical. Only pure-receiver entries may differ.
	for pid, oldPipe := range old.Service.Pipelines {
		newPipe := new.Service.Pipelines[pid]

		// Processors must be identical.
		if !slices.Equal(oldPipe.Processors, newPipe.Processors) {
			return false
		}

		// Exporters must be identical.
		if !slices.Equal(oldPipe.Exporters, newPipe.Exporters) {
			return false
		}

		// For receivers, connector entries must match.
		// Extract connector-as-receiver entries from each pipeline.
		oldConnReceivers := filterIDs(oldPipe.Receivers, isConnector)
		newConnReceivers := filterIDs(newPipe.Receivers, isConnector)
		if !slices.Equal(oldConnReceivers, newConnReceivers) {
			return false
		}
	}

	// All non-receiver aspects are identical.
	return true
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

