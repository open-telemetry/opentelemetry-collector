// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package memorylimiterprocessor // import "go.opentelemetry.io/collector/processor/memorylimiterprocessor"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/internal/memorylimiter"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
	"go.opentelemetry.io/collector/processor"
)

type memoryLimiterProcessor struct {
	memlimiter *memorylimiter.MemoryLimiter
	obsrep     *obsReport
}

// newMemoryLimiter returns a new memorylimiter processor.
func newMemoryLimiterProcessor(set processor.Settings, cfg *Config) (*memoryLimiterProcessor, error) {
	ml, err := memorylimiter.NewMemoryLimiter(cfg, set.Logger)
	if err != nil {
		return nil, err
	}
	obsrep, err := newObsReport(set)
	if err != nil {
		return nil, err
	}

	p := &memoryLimiterProcessor{
		memlimiter: ml,
		obsrep:     obsrep,
	}

	return p, nil
}

func (p *memoryLimiterProcessor) start(ctx context.Context, host component.Host) error {
	return p.memlimiter.Start(ctx, host)
}

func (p *memoryLimiterProcessor) shutdown(ctx context.Context) error {
	return p.memlimiter.Shutdown(ctx)
}

func (p *memoryLimiterProcessor) processTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	numSpans := td.SpanCount()
	if p.memlimiter.MustRefuse() {
		// TODO:
		// https://github.com/open-telemetry/opentelemetry-collector/issues/12463
		p.obsrep.refused(ctx, numSpans, pipeline.SignalTraces)
		return td, memorylimiter.ErrDataRefused
	}

	// Even if the next consumer returns error record the data as accepted by
	// this processor.
	p.obsrep.accepted(ctx, numSpans, pipeline.SignalTraces)
	return td, nil
}

func (p *memoryLimiterProcessor) processMetrics(ctx context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	numDataPoints := md.DataPointCount()
	if p.memlimiter.MustRefuse() {
		// TODO:
		// https://github.com/open-telemetry/opentelemetry-collector/issues/12463
		p.obsrep.refused(ctx, numDataPoints, pipeline.SignalMetrics)
		return md, memorylimiter.ErrDataRefused
	}

	// Even if the next consumer returns error record the data as accepted by
	// this processor.
	p.obsrep.accepted(ctx, numDataPoints, pipeline.SignalMetrics)
	return md, nil
}

func (p *memoryLimiterProcessor) processLogs(ctx context.Context, ld plog.Logs) (plog.Logs, error) {
	numRecords := ld.LogRecordCount()
	if p.memlimiter.MustRefuse() {
		// TODO:
		// https://github.com/open-telemetry/opentelemetry-collector/issues/12463
		p.obsrep.refused(ctx, numRecords, pipeline.SignalLogs)
		return ld, memorylimiter.ErrDataRefused
	}

	// Even if the next consumer returns error record the data as accepted by
	// this processor.
	p.obsrep.accepted(ctx, numRecords, pipeline.SignalLogs)
	return ld, nil
}

func (p *memoryLimiterProcessor) processProfiles(ctx context.Context, td pprofile.Profiles) (pprofile.Profiles, error) {
	numProfiles := td.SampleCount()
	if p.memlimiter.MustRefuse() {
		// TODO:
		// https://github.com/open-telemetry/opentelemetry-collector/issues/12463
		p.obsrep.refused(ctx, numProfiles, xpipeline.SignalProfiles)
		return td, memorylimiter.ErrDataRefused
	}

	// Even if the next consumer returns error record the data as accepted by
	// this processor.
	p.obsrep.accepted(ctx, numProfiles, xpipeline.SignalProfiles)
	return td, nil
}
