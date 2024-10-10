// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph // import "go.opentelemetry.io/collector/service/internal/graph"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumerprofiles"
)

// baseConsumer redeclared here since not public in consumer package. May consider to make that public.
type baseConsumer interface {
	Capabilities() consumer.Capabilities
}

type consumerNode interface {
	getConsumer() baseConsumer
}

type componentTraces struct {
	component.Component
	consumer.Traces
}

type componentMetrics struct {
	component.Component
	consumer.Metrics
}

type componentLogs struct {
	component.Component
	consumer.Logs
}

type componentProfiles struct {
	component.Component
	consumerprofiles.Profiles
}
