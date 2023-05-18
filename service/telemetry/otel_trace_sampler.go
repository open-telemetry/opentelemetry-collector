// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type recordSampler struct{}

func (r recordSampler) ShouldSample(_ sdktrace.SamplingParameters) sdktrace.SamplingResult {
	return sdktrace.SamplingResult{Decision: sdktrace.RecordOnly}
}

func (r recordSampler) Description() string {
	return "Always record sampler"
}

func alwaysRecord() sdktrace.Sampler {
	rs := &recordSampler{}
	return sdktrace.ParentBased(
		rs,
		sdktrace.WithRemoteParentSampled(sdktrace.AlwaysSample()),
		sdktrace.WithRemoteParentNotSampled(rs),
		sdktrace.WithLocalParentSampled(sdktrace.AlwaysSample()),
		sdktrace.WithRemoteParentSampled(rs))
}
