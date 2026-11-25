// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol // import "go.opentelemetry.io/collector/otelcol"

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"maps"
	"slices"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/pipeline"
)

// configFingerprint is hash-based snapshot of a configuration, used
// to detect changes without retaining or re-decoding full configs. It is built
// from the raw, pre-decode configuration map rather than decoded component.Config
// values, since those are handed to live components and may be mutated in place.
type configFingerprint struct {
	// otherHash hashes the sections that must not change: service telemetry,
	// extensions, exporters, and connectors.
	otherHash uint64

	// receiverHashes hashes each receiver's raw configuration, keyed by ID.
	receiverHashes map[component.ID]uint64

	// processorHashes hashes each processor's raw configuration, keyed by ID.
	processorHashes map[component.ID]uint64

	// extensions is the ordered list of enabled extensions. It is plain
	// plumbing, not a component.Config value handed to any component, so it
	// is safe to compare directly rather than hash.
	extensions []component.ID

	// pipelines mirrors service::pipelines. Like extensions, it is plain
	// plumbing, so it is compared directly.
	pipelines map[pipeline.ID]pipelineFingerprint
}

// pipelineFingerprint captures one pipeline's component membership.
type pipelineFingerprint struct {
	receivers  []component.ID
	processors []component.ID
	exporters  []component.ID
}

// fingerprintForPartialReload builds a configFingerprint from a configuration.
func fingerprintForPartialReload(conf *confmap.Conf, cfg *Config) (configFingerprint, error) {
	otherHash, err := hashSections(conf, "service::telemetry", "extensions", "exporters", "connectors")
	if err != nil {
		return configFingerprint{}, fmt.Errorf("could not fingerprint configuration: %w", err)
	}

	receiverHashes, err := hashEntriesByID(conf, "receivers")
	if err != nil {
		return configFingerprint{}, fmt.Errorf("could not fingerprint receiver configuration: %w", err)
	}

	processorHashes, err := hashEntriesByID(conf, "processors")
	if err != nil {
		return configFingerprint{}, fmt.Errorf("could not fingerprint processor configuration: %w", err)
	}

	pipelineFingerprints := make(map[pipeline.ID]pipelineFingerprint, len(cfg.Service.Pipelines))
	for pid, pipe := range cfg.Service.Pipelines {
		pipelineFingerprints[pid] = pipelineFingerprint{
			receivers:  slices.Clone(pipe.Receivers),
			processors: slices.Clone(pipe.Processors),
			exporters:  slices.Clone(pipe.Exporters),
		}
	}

	return configFingerprint{
		otherHash:       otherHash,
		receiverHashes:  receiverHashes,
		processorHashes: processorHashes,
		extensions:      slices.Clone(cfg.Service.Extensions),
		pipelines:       pipelineFingerprints,
	}, nil
}

// hashSections returns a stable hash over the combined raw content at the
// given confmap.Conf paths (top-level keys, or "::"-delimited nested paths
// such as "service::telemetry"). A missing path contributes a nil value, so
// its absence is still captured deterministically.
func hashSections(conf *confmap.Conf, keys ...string) (uint64, error) {
	combined := make(map[string]any, len(keys))
	for _, key := range keys {
		combined[key] = conf.Get(key)
	}
	return hashValue(combined)
}

// hashEntriesByID returns a stable hash of each entry under the given
// top-level confmap.Conf key, keyed by parsed component.ID. Used for the
// "receivers" and "processors" sections.
func hashEntriesByID(conf *confmap.Conf, key string) (map[component.ID]uint64, error) {
	raw := conf.Get(key)
	if raw == nil {
		return map[component.ID]uint64{}, nil
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected a map under %q, got %T", key, raw)
	}
	out := make(map[component.ID]uint64, len(m))
	for k, v := range m {
		var id component.ID
		if err := id.UnmarshalText([]byte(k)); err != nil {
			return nil, fmt.Errorf("invalid component id %q under %q: %w", k, key, err)
		}
		h, err := hashValue(v)
		if err != nil {
			return nil, err
		}
		out[id] = h
	}
	return out, nil
}

// hashValue returns a stable, order-independent hash of an arbitrary
// JSON-safe value, such as the maps, slices, and scalars returned by
// confmap.Conf.Get. encoding/json sorts map keys when marshaling
// map[string]any, so the hash does not depend on Go's random map iteration
// order. Unlike hashing a marshaled component.Config, this operates on the
// raw pre-decode value, so it is unaffected by configopaque.String's
// "[REDACTED]" MarshalText behavior: real secret values are hashed as-is.
func hashValue(v any) (uint64, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return 0, fmt.Errorf("could not encode value for fingerprint: %w", err)
	}
	h := fnv.New64a()
	_, _ = h.Write(b) // hash.Hash.Write never returns an error.
	return h.Sum64(), nil
}

// partialReloadSupported reports whether every difference between old and cur
// can be applied by a partial reload. Receiver differences are allowed only
// with allowReceivers, and processor differences only with allowProcessors.
// Everything else must be identical, including connector-as-receiver entries.
//
// isConnector reports whether an ID is a connector. It is only used when
// allowReceivers is set, and may be derived from either fingerprint: by the
// time it is consulted, otherHash equality has already established that the
// configured connectors are identical between old and cur.
func partialReloadSupported(old, cur configFingerprint, isConnector func(component.ID) bool, allowReceivers, allowProcessors bool) bool {
	if old.otherHash != cur.otherHash {
		return false
	}
	if !allowReceivers && !maps.Equal(old.receiverHashes, cur.receiverHashes) {
		return false
	}
	if !allowProcessors && !maps.Equal(old.processorHashes, cur.processorHashes) {
		return false
	}

	if !slices.Equal(old.extensions, cur.extensions) {
		return false
	}

	if len(old.pipelines) != len(cur.pipelines) {
		return false
	}

	for pid, oldPipe := range old.pipelines {
		curPipe, ok := cur.pipelines[pid]
		if !ok {
			return false
		}

		if !slices.Equal(oldPipe.exporters, curPipe.exporters) {
			return false
		}

		if !allowProcessors && !slices.Equal(oldPipe.processors, curPipe.processors) {
			return false
		}

		if allowReceivers {
			// Only pure-receiver entries may differ.
			oldConnReceivers := filterIDs(oldPipe.receivers, isConnector)
			curConnReceivers := filterIDs(curPipe.receivers, isConnector)
			if !slices.Equal(oldConnReceivers, curConnReceivers) {
				return false
			}
		} else if !slices.Equal(oldPipe.receivers, curPipe.receivers) {
			return false
		}
	}

	return true
}

// receiversOnlyChanged reports whether only receivers differ.
func receiversOnlyChanged(old, cur configFingerprint, isConnector func(component.ID) bool) bool {
	return partialReloadSupported(old, cur, isConnector, true, false)
}

// processorsOnlyChanged reports whether only processors differ.
func processorsOnlyChanged(old, cur configFingerprint) bool {
	return partialReloadSupported(old, cur, nil, false, true)
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

// changedReceivers returns the set of receiver IDs present in both old and
// cur whose raw configuration hash differs. IDs only used for classifying
// add/remove are intentionally omitted: graph.Update derives those
// from pipeline membership on its own, so it only ever consults this set for
// receivers it already considers "present in both".
func changedReceivers(old, cur configFingerprint) map[component.ID]bool {
	changed := make(map[component.ID]bool)
	for id, curHash := range cur.receiverHashes {
		if oldHash, ok := old.receiverHashes[id]; ok && oldHash != curHash {
			changed[id] = true
		}
	}
	return changed
}

// changedProcessors returns the processors present in both old and cur whose
// config hash differs. Additions and removals are derived from the pipeline
// processor lists.
func changedProcessors(old, cur configFingerprint) map[component.ID]bool {
	changed := make(map[component.ID]bool)
	for id, curHash := range cur.processorHashes {
		if oldHash, ok := old.processorHashes[id]; ok && oldHash != curHash {
			changed[id] = true
		}
	}
	return changed
}
