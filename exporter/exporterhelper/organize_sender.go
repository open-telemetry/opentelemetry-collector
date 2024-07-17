// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"
	"fmt"

	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

var separator = string([]byte{0x0, 0x1})

func newOrganizeSender() *organizeSender {
	return &organizeSender{}
}

type organizeSender struct {
	baseRequestSender
	organizeLogs    func(plog.Logs) []plog.Logs
	organizeMetrics func(pmetric.Metrics) []pmetric.Metrics
	organizeTraces  func(traces ptrace.Traces) []ptrace.Traces
}

func (o *organizeSender) send(ctx context.Context, req Request) error {
	var err error
	switch r := req.(type) {
	case *metricsRequest:
		if o.organizeMetrics == nil {
			return o.nextSender.send(ctx, req)
		}
		mds := (o.organizeMetrics)(r.md)
		for _, md := range mds {
			mr := &metricsRequest{
				md:     md,
				pusher: r.pusher,
			}
			err = multierr.Combine(err, o.nextSender.send(ctx, mr))
		}
	case *logsRequest:
		if o.organizeLogs == nil {
			return o.nextSender.send(ctx, req)
		}
		lds := (o.organizeLogs)(r.ld)
		for _, ld := range lds {
			lr := &logsRequest{
				ld:     ld,
				pusher: r.pusher,
			}
			err = multierr.Combine(err, o.nextSender.send(ctx, lr))
		}
	case *tracesRequest:
		if o.organizeTraces == nil {
			return o.nextSender.send(ctx, req)
		}
		tds := (o.organizeTraces)(r.td)
		for _, td := range tds {
			tr := &tracesRequest{
				td:     td,
				pusher: r.pusher,
			}
			err = multierr.Combine(err, o.nextSender.send(ctx, tr))
		}
	default:
		return fmt.Errorf("unsupported type %v", req)
	}
	return err
}
