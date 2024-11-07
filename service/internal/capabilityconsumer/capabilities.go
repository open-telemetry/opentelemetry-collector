// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package capabilityconsumer // import "go.opentelemetry.io/collector/service/internal/capabilityconsumer"

import (
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumerprofiles"
)

func NewLogs(logs consumer.Logs, cap consumer.Capabilities) consumer.Logs {
	if logs.Capabilities() == cap {
		return logs
	}
	return capLogs{Logs: logs, cap: cap}
}

type capLogs struct {
	consumer.Logs
	cap consumer.Capabilities
}

func (mts capLogs) Capabilities() consumer.Capabilities {
	return mts.cap
}

func NewMetrics(metrics consumer.Metrics, cap consumer.Capabilities) consumer.Metrics {
	if metrics.Capabilities() == cap {
		return metrics
	}
	return capMetrics{Metrics: metrics, cap: cap}
}

type capMetrics struct {
	consumer.Metrics
	cap consumer.Capabilities
}

func (mts capMetrics) Capabilities() consumer.Capabilities {
	return mts.cap
}

func NewTraces(traces consumer.Traces, cap consumer.Capabilities) consumer.Traces {
	if traces.Capabilities() == cap {
		return traces
	}
	return capTraces{Traces: traces, cap: cap}
}

type capTraces struct {
	consumer.Traces
	cap consumer.Capabilities
}

func (mts capTraces) Capabilities() consumer.Capabilities {
	return mts.cap
}

func NewProfiles(profiles consumerprofiles.Profiles, cap consumer.Capabilities) consumerprofiles.Profiles {
	if profiles.Capabilities() == cap {
		return profiles
	}
	return capProfiles{Profiles: profiles, cap: cap}
}

type capProfiles struct {
	consumerprofiles.Profiles
	cap consumer.Capabilities
}

func (mts capProfiles) Capabilities() consumer.Capabilities {
	return mts.cap
}

func NewEntities(entities consumer.Entities, cap consumer.Capabilities) consumer.Entities {
	if entities.Capabilities() == cap {
		return entities
	}
	return capEntities{Entities: entities, cap: cap}
}

type capEntities struct {
	consumer.Entities
	cap consumer.Capabilities
}

func (mts capEntities) Capabilities() consumer.Capabilities {
	return mts.cap
}
