// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package connectorexperimental // import "go.opentelemetry.io/collector/connectorexperimental"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connectorexperimental/fanoutconsumerexperimental"
	"go.opentelemetry.io/collector/consumerexperimental"
)

// ProfilesRouterAndConsumer feeds the first consumerexperimental.Profiles in each of the specified pipelines.
type ProfilesRouterAndConsumer interface {
	consumerexperimental.Profiles
	Consumer(...component.ID) (consumerexperimental.Profiles, error)
	PipelineIDs() []component.ID
	privateFunc()
}

type profilesRouter struct {
	consumerexperimental.Profiles
	baseRouter[consumerexperimental.Profiles]
}

func NewProfilesRouter(cm map[component.ID]consumerexperimental.Profiles) ProfilesRouterAndConsumer {
	consumers := make([]consumerexperimental.Profiles, 0, len(cm))
	for _, cons := range cm {
		consumers = append(consumers, cons)
	}
	return &profilesRouter{
		Profiles:   fanoutconsumerexperimental.NewProfiles(consumers),
		baseRouter: newBaseRouter(fanoutconsumerexperimental.NewProfiles, cm),
	}
}

func (r *profilesRouter) privateFunc() {}
