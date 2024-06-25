// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pipeline

import "go.opentelemetry.io/collector/component"

// InstanceID uniquely identifies a component instance
type InstanceID struct {
	ID          component.ID
	Kind        component.Kind
	PipelineIDs map[ID]struct{}
}
