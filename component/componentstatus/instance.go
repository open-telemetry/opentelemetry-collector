// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componentstatus // import "go.opentelemetry.io/collector/component/componentstatus"

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
//
// TODO: consider moving this struct to a new package/module like `extension/statuswatcher`
// https://github.com/open-telemetry/opentelemetry-collector/issues/10764
type InstanceID struct {
	componentID component.ID
	kind        component.Kind
	pipelineIDs string // IDs encoded as a string so InstanceID is Comparable.
}

// NewInstanceID returns an ID that uniquely identifies a component.
func NewInstanceID(componentID component.ID, kind component.Kind, pipelineIDs ...pipeline.ID) *InstanceID {
	instanceID := &InstanceID{
		componentID: componentID,
		kind:        kind,
	}
	instanceID.addPipelines(pipelineIDs)
	return instanceID
}

// ComponentID returns the ComponentID associated with this instance.
func (id *InstanceID) ComponentID() component.ID {
	return id.componentID
}

// Kind returns the component Kind associated with this instance.
func (id *InstanceID) Kind() component.Kind {
	return id.kind
}

// AllPipelineIDs calls f for each pipeline this instance is associated with. If
// f returns false it will stop iteration.
func (id *InstanceID) AllPipelineIDs(f func(pipeline.ID) bool) {
	var bs []byte
	for _, b := range []byte(id.pipelineIDs) {
		if b != pipelineDelim {
			bs = append(bs, b)
			continue
		}
		pipelineID := pipeline.ID{}
		err := pipelineID.UnmarshalText(bs)
		bs = bs[:0]
		if err != nil {
			continue
		}
		if !f(pipelineID) {
			break
		}
	}
}

// WithPipelines returns a new InstanceID updated to include the given
// pipelineIDs.
func (id *InstanceID) WithPipelines(pipelineIDs ...pipeline.ID) *InstanceID {
	instanceID := &InstanceID{
		componentID: id.componentID,
		kind:        id.kind,
		pipelineIDs: id.pipelineIDs,
	}
	instanceID.addPipelines(pipelineIDs)
	return instanceID
}

func (id *InstanceID) addPipelines(pipelineIDs []pipeline.ID) {
	delim := string(pipelineDelim)
	strIDs := strings.Split(id.pipelineIDs, delim)
	for _, pID := range pipelineIDs {
		strIDs = append(strIDs, pID.String())
	}
	sort.Strings(strIDs)
	strIDs = slices.Compact(strIDs)
	id.pipelineIDs = strings.Join(strIDs, delim) + delim
}
