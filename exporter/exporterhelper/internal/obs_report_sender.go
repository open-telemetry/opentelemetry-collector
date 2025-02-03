// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/internal"
)

type obsReportSender[K internal.Request] struct {
	component.StartFunc
	component.ShutdownFunc
	obsrep *ObsReport
	next   Sender[K]
}

func NewObsReportSender[K internal.Request](obsrep *ObsReport, next Sender[K]) Sender[K] {
	return &obsReportSender[K]{obsrep: obsrep, next: next}
}

func (ors *obsReportSender[K]) Send(ctx context.Context, req K) error {
	c := ors.obsrep.StartOp(ctx)
	items := req.ItemsCount()
	// Forward the data to the next consumer (this pusher is the next).
	err := ors.next.Send(c, req)
	ors.obsrep.EndOp(c, items, err)
	return err
}
