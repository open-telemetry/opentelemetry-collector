// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package capabilityconsumer // import "go.opentelemetry.io/collector/service/internal/capabilityconsumer"

import (
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
)

func NewLogs(logs consumer.Logs, capabilities consumer.Capabilities) consumer.Logs {
	if logs.Capabilities() == capabilities {
		return logs
	}
	return capLogs{Logs: logs, cap: capabilities}
}

type capLogs struct {
	consumer.Logs
	cap consumer.Capabilities
}

func (mts capLogs) Capabilities() consumer.Capabilities {
	return mts.cap
}

func NewMetrics(metrics consumer.Metrics, capabilities consumer.Capabilities) consumer.Metrics {
	if metrics.Capabilities() == capabilities {
		return metrics
	}
	return capMetrics{Metrics: metrics, cap: capabilities}
}

type capMetrics struct {
	consumer.Metrics
	cap consumer.Capabilities
}

func (mts capMetrics) Capabilities() consumer.Capabilities {
	return mts.cap
}

func NewTraces(traces consumer.Traces, capabilities consumer.Capabilities) consumer.Traces {
	if traces.Capabilities() == capabilities {
		return traces
	}
	return capTraces{Traces: traces, cap: capabilities}
}

type capTraces struct {
	consumer.Traces
	cap consumer.Capabilities
}

func (mts capTraces) Capabilities() consumer.Capabilities {
	return mts.cap
}

func NewProfiles(profiles xconsumer.Profiles, capabilities consumer.Capabilities) xconsumer.Profiles {
	if profiles.Capabilities() == capabilities {
		return profiles
	}
	return capProfiles{Profiles: profiles, cap: capabilities}
}

type capProfiles struct {
	xconsumer.Profiles
	cap consumer.Capabilities
}

func (mts capProfiles) Capabilities() consumer.Capabilities {
	return mts.cap
}
