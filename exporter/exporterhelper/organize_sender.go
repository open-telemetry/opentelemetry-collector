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

func newOrganizeSender(attrKeys []string) *organizeSender {
	return &organizeSender{
		attrKeys: attrKeys,
	}
}

type organizeSender struct {
	baseRequestSender
	attrKeys []string
}

func (o *organizeSender) send(ctx context.Context, req Request) error {
	var err error
	switch r := req.(type) {
	case *metricsRequest:
		mds := organizeMetrics(o.attrKeys, r.md)
		for _, md := range mds {
			mr := &metricsRequest{
				md:     md,
				pusher: r.pusher,
			}
			err = multierr.Combine(err, o.nextSender.send(ctx, mr))
		}
	case *logsRequest:
		lds := organizeLogs(o.attrKeys, r.ld)
		for _, ld := range lds {
			lr := &logsRequest{
				ld:     ld,
				pusher: r.pusher,
			}
			err = multierr.Combine(err, o.nextSender.send(ctx, lr))
		}
	case *tracesRequest:
		tds := organizeTraces(o.attrKeys, r.td)
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

func organizeLogs(attrKeys []string, ld plog.Logs) []plog.Logs {
	rls := ld.ResourceLogs()
	lenRls := rls.Len()
	// If zero or one resource logs, return early
	if lenRls <= 1 {
		return []plog.Logs{ld}
	}

	indicesByAttr := make(map[string][]int)
	for i := 0; i < lenRls; i++ {
		rl := rls.At(i)
		var attrVal string
		for _, k := range attrKeys {
			if attributeValue, ok := rl.Resource().Attributes().Get(k); ok {
				attrVal = fmt.Sprintf("%s%s%s", attrVal, separator, attributeValue.Str())
			}
		}
		indicesByAttr[attrVal] = append(indicesByAttr[attrVal], i)
	}
	// If there is a single attribute value, return early
	if len(indicesByAttr) <= 1 {
		return []plog.Logs{ld}
	}

	var result []plog.Logs
	for _, indices := range indicesByAttr {
		logsForAttr := plog.NewLogs()
		result = append(result, logsForAttr)
		for _, i := range indices {
			rs := rls.At(i)
			rs.CopyTo(logsForAttr.ResourceLogs().AppendEmpty())
		}
	}
	return result
}

func organizeMetrics(attrKeys []string, md pmetric.Metrics) []pmetric.Metrics {
	rms := md.ResourceMetrics()
	lenRls := rms.Len()
	// If zero or one resource metrics, return early
	if lenRls <= 1 {
		return []pmetric.Metrics{md}
	}

	indicesByAttr := make(map[string][]int)
	for i := 0; i < lenRls; i++ {
		rm := rms.At(i)
		var attrVal string
		for _, k := range attrKeys {
			if attributeValue, ok := rm.Resource().Attributes().Get(k); ok {
				attrVal = fmt.Sprintf("%s%s%s", attrVal, separator, attributeValue.Str())
			}
		}
		indicesByAttr[attrVal] = append(indicesByAttr[attrVal], i)
	}
	// If there is a single attribute value, return early
	if len(indicesByAttr) <= 1 {
		return []pmetric.Metrics{md}
	}

	var result []pmetric.Metrics
	for _, indices := range indicesByAttr {
		metricsForAttr := pmetric.NewMetrics()
		result = append(result, metricsForAttr)
		for _, i := range indices {
			rs := rms.At(i)
			rs.CopyTo(metricsForAttr.ResourceMetrics().AppendEmpty())
		}
	}
	return result
}

func organizeTraces(attrKeys []string, td ptrace.Traces) []ptrace.Traces {
	rss := td.ResourceSpans()
	lenRls := rss.Len()
	// If zero or one resource traces, return early
	if lenRls <= 1 {
		return []ptrace.Traces{td}
	}

	indicesByAttr := make(map[string][]int)
	for i := 0; i < lenRls; i++ {
		rs := rss.At(i)
		var attrVal string
		for _, k := range attrKeys {
			if attributeValue, ok := rs.Resource().Attributes().Get(k); ok {
				attrVal = fmt.Sprintf("%s%s%s", attrVal, separator, attributeValue.Str())
			}
		}
		indicesByAttr[attrVal] = append(indicesByAttr[attrVal], i)
	}
	// If there is a single attribute value, return early
	if len(indicesByAttr) <= 1 {
		return []ptrace.Traces{td}
	}

	var result []ptrace.Traces
	for _, indices := range indicesByAttr {
		tracesForAttr := ptrace.NewTraces()
		result = append(result, tracesForAttr)
		for _, i := range indices {
			rs := rss.At(i)
			rs.CopyTo(tracesForAttr.ResourceSpans().AppendEmpty())
		}
	}
	return result
}
