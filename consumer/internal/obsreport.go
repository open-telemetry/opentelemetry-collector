// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/consumer/internal"

import "context"

// ObsReport contains information required to make an implementor
// of Consumer observable.
type ObsReport interface {
	StartTracesOp(context.Context) context.Context
	EndTracesOp(context.Context, int, error)
}

type baseObsReport struct{}

func (bor baseObsReport) StartTracesOp(ctx context.Context) context.Context {
	return ctx
}

func (bor baseObsReport) EndTracesOp(_ context.Context, _ int, _ error) {}

var noopObsReport = baseObsReport{}
