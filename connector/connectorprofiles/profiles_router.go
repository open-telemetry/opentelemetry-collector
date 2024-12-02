// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package connectorprofiles // import "go.opentelemetry.io/collector/connector/connectorprofiles"

import (
	"go.opentelemetry.io/collector/connector/internal"
	"go.opentelemetry.io/collector/consumer/consumerexp"
	"go.opentelemetry.io/collector/internal/fanoutconsumer"
	"go.opentelemetry.io/collector/pipeline"
)

type ProfilesRouterAndConsumer interface {
	consumerexp.Profiles
	Consumer(...pipeline.ID) (consumerexp.Profiles, error)
	PipelineIDs() []pipeline.ID
	privateFunc()
}

type profilesRouter struct {
	consumerexp.Profiles
	internal.BaseRouter[consumerexp.Profiles]
}

func NewProfilesRouter(cm map[pipeline.ID]consumerexp.Profiles) ProfilesRouterAndConsumer {
	consumers := make([]consumerexp.Profiles, 0, len(cm))
	for _, cons := range cm {
		consumers = append(consumers, cons)
	}
	return &profilesRouter{
		Profiles:   fanoutconsumer.NewProfiles(consumers),
		BaseRouter: internal.NewBaseRouter(fanoutconsumer.NewProfiles, cm),
	}
}

func (r *profilesRouter) privateFunc() {}
