// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"

	"go.opentelemetry.io/collector/exporter/internal"
)

type obsReportSender[K internal.Request] struct {
	BaseSender[K]
	obsrep *ObsReport
}

func NewObsReportSender[K internal.Request](obsrep *ObsReport) Sender[K] {
	return &obsReportSender[K]{obsrep: obsrep}
}

func (ors *obsReportSender[K]) Send(ctx context.Context, req K) error {
	c := ors.obsrep.StartOp(ctx)
	items := req.ItemsCount()
	// Forward the data to the next consumer (this pusher is the next).
	err := ors.NextSender.Send(c, req)
	ors.obsrep.EndOp(c, items, err)
	return err
}
