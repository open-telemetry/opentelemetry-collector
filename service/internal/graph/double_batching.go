// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph // import "go.opentelemetry.io/collector/service/internal/graph"

import (
	"reflect"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/service/internal/builders"
	"go.opentelemetry.io/collector/service/pipelines"
)

const batchProcessorType = "batch"

// warnDoubleBatching inspects the active execution graph and emits a warning for
// every (pipeline, exporter) pair where both conditions hold:
//
//  1. The pipeline contains a batchprocessor (type "batch") in its processor chain.
//  2. The exporter's config declares a sending_queue section whose batch sub-section
//     is present (i.e. exporterhelper batching is enabled on that exporter).
//
// Only pipelines that are actually reachable in the execution graph are examined,
// so connectors, disabled components, and components that exist in config but are
// not wired into a pipeline path do not trigger false positives.
func warnDoubleBatching(
	logger *zap.Logger,
	pipelineConfigs pipelines.Config,
	exporterBuilder *builders.ExporterBuilder,
) {
	for pipelineID, pipelineCfg := range pipelineConfigs {
		// Collect any batch processor component IDs in this pipeline.
		var batchProcIDs []component.ID
		for _, procID := range pipelineCfg.Processors {
			if procID.Type().String() == batchProcessorType {
				batchProcIDs = append(batchProcIDs, procID)
			}
		}
		if len(batchProcIDs) == 0 {
			continue
		}

		// For each exporter wired into this pipeline, check if it has batching enabled.
		for _, exporterID := range pipelineCfg.Exporters {
			cfg := exporterBuilder.Config(exporterID)
			if cfg == nil {
				continue
			}
			if !exporterHasBatchingEnabled(cfg) {
				continue
			}

			// Build a human-readable path description.
			path := buildDoubleBatchingPath(pipelineID, batchProcIDs, exporterID)
			logger.Warn(
				"double batching detected: both a batch processor and exporterhelper batching are active on the same signal path; "+
					"remove the batch processor or disable exporterhelper batching to avoid paying batching overhead twice",
				zap.String("pipeline", pipelineID.String()),
				zap.String("exporter", exporterID.String()),
				zap.String("path", path),
			)
		}
	}
}

// exporterHasBatchingEnabled uses reflection to determine whether the given
// component.Config has exporterhelper batching enabled.
//
// It looks for a struct field with mapstructure tag "sending_queue" that itself
// exposes a HasValue() bool method.  If that optional has a value, it then looks
// for a nested field with mapstructure tag "batch" that also exposes HasValue().
// Only when both HasValue() calls return true is the exporter considered to have
// batching enabled.
func exporterHasBatchingEnabled(cfg component.Config) bool {
	v := reflect.ValueOf(cfg)
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return false
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return false
	}

	// Find the sending_queue field.
	queueVal, ok := findFieldByTag(v, "sending_queue")
	if !ok {
		return false
	}
	if !callHasValue(queueVal) {
		return false
	}

	// The sending_queue field itself has a value; now look for the batch sub-field.
	// QueueBatchConfig embeds configoptional.Optional[BatchConfig] as field "batch".
	// We need to dereference into the inner value first.
	innerVal := resolveOptionalValue(queueVal)
	if !innerVal.IsValid() {
		return false
	}
	if innerVal.Kind() == reflect.Pointer {
		if innerVal.IsNil() {
			return false
		}
		innerVal = innerVal.Elem()
	}
	if innerVal.Kind() != reflect.Struct {
		return false
	}

	batchVal, ok := findFieldByTag(innerVal, "batch")
	if !ok {
		return false
	}
	return callHasValue(batchVal)
}

// findFieldByTag finds the first struct field whose mapstructure tag matches tag.
func findFieldByTag(v reflect.Value, tag string) (reflect.Value, bool) {
	t := v.Type()
	for i := range t.NumField() {
		ft := t.Field(i)
		ms := ft.Tag.Get("mapstructure")
		// mapstructure tags may have options after a comma (e.g. "name,omitempty").
		name := ms
		if idx := len(ms); idx > 0 {
			for j, c := range ms {
				if c == ',' {
					name = ms[:j]
					break
				}
			}
		}
		if name == tag {
			return v.Field(i), true
		}
	}
	return reflect.Value{}, false
}

// callHasValue calls HasValue() on v if v has that method, and returns the bool result.
func callHasValue(v reflect.Value) bool {
	// HasValue is defined on the value receiver of Optional[T], so we call it on
	// the addressable copy when necessary.
	m := v.MethodByName("HasValue")
	if !m.IsValid() {
		// Try pointer receiver.
		if v.CanAddr() {
			m = v.Addr().MethodByName("HasValue")
		}
	}
	if !m.IsValid() {
		return false
	}
	results := m.Call(nil)
	if len(results) != 1 || results[0].Kind() != reflect.Bool {
		return false
	}
	return results[0].Bool()
}

// resolveOptionalValue calls Get() on an Optional[T] value and dereferences the
// returned pointer so that callers receive the underlying T value.
func resolveOptionalValue(v reflect.Value) reflect.Value {
	m := v.MethodByName("Get")
	if !m.IsValid() {
		if v.CanAddr() {
			m = v.Addr().MethodByName("Get")
		}
	}
	if !m.IsValid() {
		return reflect.Value{}
	}
	results := m.Call(nil)
	if len(results) == 1 && results[0].Kind() == reflect.Pointer && !results[0].IsNil() {
		return results[0].Elem()
	}
	return reflect.Value{}
}

// buildDoubleBatchingPath constructs a readable path like:
//
//	traces[/name]: batchprocessor[/name] -> exporter/otlp
func buildDoubleBatchingPath(pipelineID pipeline.ID, batchProcs []component.ID, exporterID component.ID) string {
	procs := ""
	for i, p := range batchProcs {
		if i > 0 {
			procs += ", "
		}
		procs += p.String()
	}
	return pipelineID.String() + ": processor/" + procs + " -> exporter/" + exporterID.String()
}
