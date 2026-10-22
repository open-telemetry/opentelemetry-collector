// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package sampleprocessor // import "go.opentelemetry.io/collector/cmd/mdatagen/internal/sampleprocessor"

import (
	"context"

	"go.opentelemetry.io/collector/cmd/mdatagen/internal/sampleprocessor/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/xprocessor"
)

// NewFactory returns a receiver.Factory for sample receiver.
func NewFactory() processor.Factory {
	return xprocessor.NewFactory(
		metadata.Type,
		func() component.Config { return &struct{}{} },
		xprocessor.WithTraces(createTracesProcessor, metadata.TracesStability),
		xprocessor.WithMetrics(createMetricsProcessor, metadata.MetricsStability),
		xprocessor.WithLogs(createLogsProcessor, metadata.LogsStability),
		xprocessor.WithProfiles(createProfilesProcessor, metadata.ProfilesStability),
	)
}

func createTracesProcessor(_ context.Context, _ processor.Settings, _ component.Config, next consumer.Traces) (processor.Traces, error) {
	return &nopProcessor{nextTraces: next}, nil
}

func createMetricsProcessor(_ context.Context, _ processor.Settings, _ component.Config, next consumer.Metrics) (processor.Metrics, error) {
	return &nopProcessor{nextMetrics: next}, nil
}

func createLogsProcessor(_ context.Context, _ processor.Settings, _ component.Config, next consumer.Logs) (processor.Logs, error) {
	return &nopProcessor{nextLogs: next}, nil
}

func createProfilesProcessor(_ context.Context, _ processor.Settings, _ component.Config, next xconsumer.Profiles) (xprocessor.Profiles, error) {
	return &nopProcessor{nextProfiles: next}, nil
}

type nopProcessor struct {
	component.StartFunc
	component.ShutdownFunc
	nextTraces   consumer.Traces
	nextMetrics  consumer.Metrics
	nextLogs     consumer.Logs
	nextProfiles xconsumer.Profiles
}

func (n *nopProcessor) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	return n.nextTraces.ConsumeTraces(ctx, td)
}

func (n *nopProcessor) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	return n.nextLogs.ConsumeLogs(ctx, ld)
}

func (*nopProcessor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

func (n *nopProcessor) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	return n.nextMetrics.ConsumeMetrics(ctx, md)
}

func (n *nopProcessor) ConsumeProfiles(ctx context.Context, pd pprofile.Profiles) error {
	return n.nextProfiles.ConsumeProfiles(ctx, pd)
}
