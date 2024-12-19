// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xconnector // import "go.opentelemetry.io/collector/connector/xconnector"

import (
	"go.opentelemetry.io/collector/connector/internal"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/internal/fanoutconsumer"
	"go.opentelemetry.io/collector/pipeline"
)

type ProfilesRouterAndConsumer interface {
	xconsumer.Profiles
	Consumer(...pipeline.ID) (xconsumer.Profiles, error)
	PipelineIDs() []pipeline.ID
	privateFunc()
}

type profilesRouter struct {
	xconsumer.Profiles
	internal.BaseRouter[xconsumer.Profiles]
}

func NewProfilesRouter(cm map[pipeline.ID]xconsumer.Profiles) ProfilesRouterAndConsumer {
	consumers := make([]xconsumer.Profiles, 0, len(cm))
	for _, cons := range cm {
		consumers = append(consumers, cons)
	}
	return &profilesRouter{
		Profiles:   fanoutconsumer.NewProfiles(consumers),
		BaseRouter: internal.NewBaseRouter(fanoutconsumer.NewProfiles, cm),
	}
}

func (r *profilesRouter) privateFunc() {}
