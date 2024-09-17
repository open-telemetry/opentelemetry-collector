// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package connectorprofiles // import "go.opentelemetry.io/collector/connector/connectorprofiles"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector/internal"
	"go.opentelemetry.io/collector/consumer/consumerprofiles"
	"go.opentelemetry.io/collector/internal/fanoutconsumer"
	"go.opentelemetry.io/collector/pipeline"
)

// ProfilesRouterAndConsumer feeds the first consumerprofiles.Profiles in each of the specified pipelines.
//
// Deprecated [v0.110.0] Use ProfilesRouterAndConsumerWithPipelineIDs instead
type ProfilesRouterAndConsumer interface {
	consumerprofiles.Profiles
	Consumer(...component.ID) (consumerprofiles.Profiles, error)
	PipelineIDs() []component.ID
	privateFunc()
}

type ProfilesRouterAndConsumerWithPipelineIDs interface {
	consumerprofiles.Profiles
	Consumer(...pipeline.ID) (consumerprofiles.Profiles, error)
	PipelineIDs() []pipeline.ID
	privateFunc()
}

type profilesRouter struct {
	consumerprofiles.Profiles
	internal.BaseRouter[consumerprofiles.Profiles]
}

// Deprecated: [v0.110.0] Use NewProfilesRouterWithPipelineIDs instead.
func NewProfilesRouter(cm map[component.ID]consumerprofiles.Profiles) ProfilesRouterAndConsumer {
	consumers := make([]consumerprofiles.Profiles, 0, len(cm))
	for _, cons := range cm {
		consumers = append(consumers, cons)
	}
	return &profilesRouter{
		Profiles:   fanoutconsumer.NewProfiles(consumers),
		BaseRouter: internal.NewBaseRouter(fanoutconsumer.NewProfiles, cm),
	}
}

func (r *profilesRouter) privateFunc() {}

type profilesRouterPipelineIDs struct {
	consumerprofiles.Profiles
	internal.BaseRouterWithPipelineIDs[consumerprofiles.Profiles]
}

func NewProfilesRouterWithPipelineIDs(cm map[pipeline.ID]consumerprofiles.Profiles) ProfilesRouterAndConsumerWithPipelineIDs {
	consumers := make([]consumerprofiles.Profiles, 0, len(cm))
	for _, cons := range cm {
		consumers = append(consumers, cons)
	}
	return &profilesRouterPipelineIDs{
		Profiles:                  fanoutconsumer.NewProfiles(consumers),
		BaseRouterWithPipelineIDs: internal.NewBaseRouterWithPipelineIDs(fanoutconsumer.NewProfiles, cm),
	}
}

func (r *profilesRouterPipelineIDs) privateFunc() {}
