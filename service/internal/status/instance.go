// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package status // import "go.opentelemetry.io/collector/service/internal/status"

import (
	"slices"
	"sort"
	"strings"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pipeline"
)

// pipelineDelim is the delimiter for internal representation of pipeline
// component IDs.
const pipelineDelim = byte(0x20)

// InstanceID uniquely identifies a component instance
type InstanceID struct {
	ComponentID component.ID
	Kind        component.Kind
	pipelineIDs string // IDs encoded as a string so InstanceID is Comparable.
}

// NewInstanceID returns an ID that uniquely identifies a component.
func NewInstanceID(componentID component.ID, kind component.Kind, pipelineIDs ...pipeline.ID) *InstanceID {
	instanceID := &InstanceID{
		ComponentID: componentID,
		Kind:        kind,
	}
	instanceID.addPipelines(pipelineIDs)
	return instanceID
}

func (id *InstanceID) PipelineIDs() []pipeline.ID {
	strIDs := strings.Split(id.pipelineIDs, string(pipelineDelim))

	ids := make([]pipeline.ID, len(strIDs))
	for i, strID := range strIDs {
		pipelineID := pipeline.ID{}
		_ = pipelineID.UnmarshalText([]byte(strID))
		ids[i] = pipelineID
	}

	return ids
}

// WithPipelines returns a new InstanceID updated to include the given
// pipelineIDs.
func (id *InstanceID) WithPipelines(pipelineIDs ...pipeline.ID) *InstanceID {
	instanceID := &InstanceID{
		ComponentID: id.ComponentID,
		Kind:        id.Kind,
		pipelineIDs: id.pipelineIDs,
	}
	instanceID.addPipelines(pipelineIDs)
	return instanceID
}

func (id *InstanceID) addPipelines(pipelineIDs []pipeline.ID) {
	delim := string(pipelineDelim)
	var strIDs []string
	if id.pipelineIDs != "" {
		strIDs = strings.Split(id.pipelineIDs, delim)
	}
	for _, pID := range pipelineIDs {
		strIDs = append(strIDs, pID.String())
	}
	sort.Strings(strIDs)
	strIDs = slices.Compact(strIDs)
	id.pipelineIDs = strings.Join(strIDs, delim)
}
