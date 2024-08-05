// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componentstatus // import "go.opentelemetry.io/collector/component/componentstatus"

import "go.opentelemetry.io/collector/component"

// InstanceID uniquely identifies a component instance
//
// TODO: consider moving this struct to a new package/module like `extension/statuswatcher`
// https://github.com/open-telemetry/opentelemetry-collector/issues/10764
type InstanceID struct {
	ID          component.ID
	Kind        component.Kind
	PipelineIDs map[component.ID]struct{}
}
