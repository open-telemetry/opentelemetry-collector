// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xconnector // import "go.opentelemetry.io/collector/connector/xconnector"

import (
	"go.opentelemetry.io/collector/connector/internal"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/internal/fanoutconsumer"
	"go.opentelemetry.io/collector/pipeline"
)

// EntitiesRouterAndConsumer feeds the first xconsumer.Entities in each of the specified pipelines.
type EntitiesRouterAndConsumer interface {
	xconsumer.Entities
	Consumer(...pipeline.ID) (xconsumer.Entities, error)
	PipelineIDs() []pipeline.ID
	privateFunc()
}

type entitiesRouter struct {
	xconsumer.Entities
	internal.BaseRouter[xconsumer.Entities]
}

func NewEntitiesRouter(cm map[pipeline.ID]xconsumer.Entities) EntitiesRouterAndConsumer {
	consumers := make([]xconsumer.Entities, 0, len(cm))
	for _, cons := range cm {
		consumers = append(consumers, cons)
	}
	return &entitiesRouter{
		Entities:   fanoutconsumer.NewEntities(consumers),
		BaseRouter: internal.NewBaseRouter(fanoutconsumer.NewEntities, cm),
	}
}

func (r *entitiesRouter) privateFunc() {}
