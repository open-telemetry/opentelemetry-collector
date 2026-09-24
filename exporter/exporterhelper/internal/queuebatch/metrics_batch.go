// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sizer"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

// MergeSplit splits and/or merges the provided metrics request and the current request into one or more requests
// conforming with the MaxSizeConfig.
func (req *metricsRequest) MergeSplit(_ context.Context, maxSize int, szt request.SizerType, r2 request.Request) ([]request.Request, error) {
	var sz sizer.MetricsSizer
	switch szt {
	case request.SizerTypeItems:
		sz = &sizer.MetricsCountSizer{}
	case request.SizerTypeBytes:
		sz = &sizer.MetricsBytesSizer{}
	default:
		return nil, errors.New("unknown sizer type")
	}

	if r2 != nil {
		req2, ok := r2.(*metricsRequest)
		if !ok {
			return nil, errors.New("invalid input type")
		}
		req2.mergeTo(req, sz, szt)
	}

	// If no limit we can simply merge the new request into the current and return.
	if maxSize == 0 {
		return []request.Request{req}, nil
	}
	return req.split(maxSize, sz, szt)
}

func (req *metricsRequest) mergeTo(dst *metricsRequest, sz sizer.MetricsSizer, szt request.SizerType) {
	if sz != nil {
		dst.sizes.Update(szt, dst.size(sz, szt)+req.size(sz, szt))
		req.sizes.Update(szt, 0)
	}
	req.md.ResourceMetrics().MoveAndAppendTo(dst.md.ResourceMetrics())
}

func (req *metricsRequest) split(maxSize int, sz sizer.MetricsSizer, szt request.SizerType) ([]request.Request, error) {
	var res []request.Request
	droppedItems := 0
	pruned := false
	unsplittable := false
	for req.size(sz, szt) > maxSize {
		md, rmSize := extractMetrics(req.md, maxSize, sz)
		if md.DataPointCount() == 0 {
			if metricsCarryContent(md) {
				// Extraction moved metrics that hold no data point out of the source. They
				// still carry a name, unit and description, which dropOversizedDataPoints
				// takes care to preserve, so the batch goes out rather than being discarded.
				// The rmSize accounting assumes data points moved, so recompute exactly.
				req.sizes.Update(szt, sz.MetricsSize(req.md))
				res = append(res, newMetricsRequest(md))
				continue
			}
			if md.ResourceMetrics().Len() > 0 {
				// Extraction spent this batch on resources and scopes that hold no metric at
				// all and the batch is discarded with them, but they left the source as it
				// did so. The request is smaller than when this attempt started, so try again
				// with a fresh batch instead of dropping anything: the next data point may
				// well fit.
				req.sizes.Update(szt, sz.MetricsSize(req.md))
				continue
			}
			// The next data point does not fit into maxSize even on its own. Drop every
			// data point in that state in a single pass, rather than one per iteration
			// with a full size recompute after each, then carry on splitting what is left.
			if !pruned {
				pruned = true
				var removedAny bool
				droppedItems, removedAny = dropOversizedDataPoints(req.md, maxSize, sz)
				// Refresh the cache whether or not a data point went: the pass also prunes
				// the metrics, scopes and resources it empties, which changes the size on
				// its own.
				req.sizes.Update(szt, sz.MetricsSize(req.md))
				if removedAny {
					continue
				}
			}
			// Nothing was extracted, nothing left the source and the pass found nothing
			// to remove, so no further progress is possible. Stop rather than loop.
			unsplittable = true
			break
		}
		req.sizes.Update(szt, req.size(sz, szt)-rmSize)
		res = append(res, newMetricsRequest(md))
	}
	// Keep the remainder, unless the drop pass emptied it, in which case there is
	// nothing left to export. A metric that holds no data point still counts as
	// something to export, so the test is whether a metric is left rather than a
	// data point.
	if !pruned || metricsCarryContent(req.md) {
		res = append(res, req)
	}
	switch {
	case droppedItems > 0:
		return res, fmt.Errorf("one datapoint size is greater than max size, dropping items: %d", droppedItems)
	case unsplittable && metricsCarryContent(req.md):
		// Only worth reporting when a remainder is going out unsplit. With nothing left
		// in it there is no metric to lose and nothing to tell the caller.
		return res, errors.New("request size is greater than max size and cannot be split further")
	}
	return res, nil
}

// metricsCarryContent reports whether md holds at least one metric.
//
// A metric that carries no data point still describes a measurement through its
// name, unit and description, so DataPointCount alone cannot decide whether a batch
// is worth exporting: dropOversizedDataPoints deliberately keeps such a metric, and
// discarding the batch that holds it would undo that. Resources and scopes holding
// no metric carry no measurement and do not count.
func metricsCarryContent(md pmetric.Metrics) bool {
	for i := range md.ResourceMetrics().Len() {
		rm := md.ResourceMetrics().At(i)
		for j := range rm.ScopeMetrics().Len() {
			if rm.ScopeMetrics().At(j).Metrics().Len() > 0 {
				return true
			}
		}
	}
	return false
}

// dropOversizedDataPoints removes every data point that cannot fit a batch of
// maxSize even on its own, together with the metric, scope and resource it leaves
// empty, and reports how many it removed.
//
// A data point shares each batch with its metric, resource and scope, so their
// framing counts against maxSize too. The capacity left for a data point is
// therefore computed the same way extractResourceMetrics and extractScopeMetrics
// compute it, which keeps this pass in step with what extraction would accept.
//
// A metric holding no data points is left alone: there is nothing in it to drop, so
// removing it would lose its name, unit and description without counting an item.
// Such a metric is removed only when it cannot be exported at all, which is why the
// caller is told separately whether anything was removed: that case makes progress
// without dropping a single item.
func dropOversizedDataPoints(md pmetric.Metrics, maxSize int, sz sizer.MetricsSizer) (droppedItems int, removedAny bool) {
	dropped := 0
	batchCapacity := maxSize - sz.MetricsSize(pmetric.NewMetrics())
	md.ResourceMetrics().RemoveIf(func(rm pmetric.ResourceMetrics) bool {
		bareRM := pmetric.NewResourceMetrics()
		bareRM.SetSchemaUrl(rm.SchemaUrl())
		rm.Resource().CopyTo(bareRM.Resource())
		scopeCapacity := batchCapacity - (sz.DeltaSize(batchCapacity) - batchCapacity) - sz.ResourceMetricsSize(bareRM)
		rm.ScopeMetrics().RemoveIf(func(sm pmetric.ScopeMetrics) bool {
			bareSM := pmetric.NewScopeMetrics()
			bareSM.SetSchemaUrl(sm.SchemaUrl())
			sm.Scope().CopyTo(bareSM.Scope())
			metricCapacity := scopeCapacity - (sz.DeltaSize(scopeCapacity) - scopeCapacity) - sz.ScopeMetricsSize(bareSM)
			sm.Metrics().RemoveIf(func(m pmetric.Metric) bool {
				had := dataPointsLen(m)
				n := dropOversizedMetricDataPoints(m, metricCapacity, sz)
				dropped += n
				if n > 0 {
					removedAny = true
				}
				if had > 0 {
					// Drop the metric once every data point it held turned out oversized.
					return dataPointsLen(m) == 0
				}
				// A metric that never carried a data point holds no item to drop, so it is
				// kept, unless its own framing cannot fit a batch either. Such a metric can
				// never be exported and would block every data point behind it, so drop it
				// without counting an item: no measurement is lost, only its name, unit and
				// description.
				if sz.DeltaSize(sz.MetricSize(m)) > metricCapacity {
					removedAny = true
					return true
				}
				return false
			})
			if sm.Metrics().Len() == 0 {
				removedAny = true
				return true
			}
			return false
		})
		if rm.ScopeMetrics().Len() == 0 {
			removedAny = true
			return true
		}
		return false
	})
	return dropped, removedAny
}

// dropOversizedMetricDataPoints removes the data points of m that cannot fit the
// given capacity, whichever data point type m holds, and reports how many it
// removed.
//
// A data point is framed by its own metric as well, so the metric's name, unit,
// description and metadata come off the capacity before a data point is measured,
// the way extractMetricDataPoints and the extract*DataPoints helpers do it.
func dropOversizedMetricDataPoints(m pmetric.Metric, capacity int, sz sizer.MetricsSizer) int {
	bare := pmetric.NewMetric()
	bare.SetName(m.Name())
	bare.SetDescription(m.Description())
	bare.SetUnit(m.Unit())
	m.Metadata().CopyTo(bare.Metadata())

	dropped := 0
	switch m.Type() {
	case pmetric.MetricTypeEmpty:
		// No data point slice exists on this metric, so there is nothing to remove.
	case pmetric.MetricTypeGauge:
		bare.SetEmptyGauge()
		dpCapacity := capacity - (sz.DeltaSize(capacity) - capacity) - sz.MetricSize(bare)
		m.Gauge().DataPoints().RemoveIf(func(dp pmetric.NumberDataPoint) bool {
			if sz.DeltaSize(sz.NumberDataPointSize(dp)) > dpCapacity {
				dropped++
				return true
			}
			return false
		})
	case pmetric.MetricTypeSum:
		bare.SetEmptySum()
		dpCapacity := capacity - (sz.DeltaSize(capacity) - capacity) - sz.MetricSize(bare)
		m.Sum().DataPoints().RemoveIf(func(dp pmetric.NumberDataPoint) bool {
			if sz.DeltaSize(sz.NumberDataPointSize(dp)) > dpCapacity {
				dropped++
				return true
			}
			return false
		})
	case pmetric.MetricTypeHistogram:
		bare.SetEmptyHistogram()
		dpCapacity := capacity - (sz.DeltaSize(capacity) - capacity) - sz.MetricSize(bare)
		m.Histogram().DataPoints().RemoveIf(func(dp pmetric.HistogramDataPoint) bool {
			if sz.DeltaSize(sz.HistogramDataPointSize(dp)) > dpCapacity {
				dropped++
				return true
			}
			return false
		})
	case pmetric.MetricTypeExponentialHistogram:
		bare.SetEmptyExponentialHistogram()
		dpCapacity := capacity - (sz.DeltaSize(capacity) - capacity) - sz.MetricSize(bare)
		m.ExponentialHistogram().DataPoints().RemoveIf(func(dp pmetric.ExponentialHistogramDataPoint) bool {
			if sz.DeltaSize(sz.ExponentialHistogramDataPointSize(dp)) > dpCapacity {
				dropped++
				return true
			}
			return false
		})
	case pmetric.MetricTypeSummary:
		bare.SetEmptySummary()
		dpCapacity := capacity - (sz.DeltaSize(capacity) - capacity) - sz.MetricSize(bare)
		m.Summary().DataPoints().RemoveIf(func(dp pmetric.SummaryDataPoint) bool {
			if sz.DeltaSize(sz.SummaryDataPointSize(dp)) > dpCapacity {
				dropped++
				return true
			}
			return false
		})
	}
	return dropped
}

// extractMetrics extracts metrics from srcMetrics until capacity is reached.
func extractMetrics(srcMetrics pmetric.Metrics, capacity int, sz sizer.MetricsSizer) (pmetric.Metrics, int) {
	destMetrics := pmetric.NewMetrics()
	capacityLeft := capacity - sz.MetricsSize(destMetrics)
	removedSize := 0
	srcMetrics.ResourceMetrics().RemoveIf(func(srcRM pmetric.ResourceMetrics) bool {
		// If the no more capacity left just return.
		if capacityLeft == 0 {
			return false
		}
		rawRlSize := sz.ResourceMetricsSize(srcRM)
		rlSize := sz.DeltaSize(rawRlSize)
		if rlSize > capacityLeft {
			extSrcRM, extRmSize := extractResourceMetrics(srcRM, capacityLeft, sz)
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			removedSize += extRmSize
			// There represents the delta between the delta sizes.
			removedSize += rlSize - rawRlSize - (sz.DeltaSize(rawRlSize-extRmSize) - (rawRlSize - extRmSize))
			// It is possible that for the bytes scenario, the extracted field contains no scope metrics.
			// Do not add it to the destination if that is the case.
			if extSrcRM.ScopeMetrics().Len() > 0 {
				extSrcRM.MoveTo(destMetrics.ResourceMetrics().AppendEmpty())
			}
			return extSrcRM.ScopeMetrics().Len() != 0
		}
		capacityLeft -= rlSize
		removedSize += rlSize
		srcRM.MoveTo(destMetrics.ResourceMetrics().AppendEmpty())
		return true
	})
	return destMetrics, removedSize
}

// extractResourceMetrics extracts resource metrics and returns a new resource metrics with the specified number of data points.
func extractResourceMetrics(srcRM pmetric.ResourceMetrics, capacity int, sz sizer.MetricsSizer) (pmetric.ResourceMetrics, int) {
	destRM := pmetric.NewResourceMetrics()
	destRM.SetSchemaUrl(srcRM.SchemaUrl())
	srcRM.Resource().CopyTo(destRM.Resource())
	// Take into account that this can have max "capacity", so when added to the parent will need space for the extra delta size.
	capacityLeft := capacity - (sz.DeltaSize(capacity) - capacity) - sz.ResourceMetricsSize(destRM)
	removedSize := 0
	srcRM.ScopeMetrics().RemoveIf(func(srcSM pmetric.ScopeMetrics) bool {
		// If the no more capacity left just return.
		if capacityLeft == 0 {
			return false
		}
		rawSmSize := sz.ScopeMetricsSize(srcSM)
		smSize := sz.DeltaSize(rawSmSize)
		if smSize > capacityLeft {
			extSrcSM, extSmSize := extractScopeMetrics(srcSM, capacityLeft, sz)
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			removedSize += extSmSize
			// There represents the delta between the delta sizes.
			removedSize += smSize - rawSmSize - (sz.DeltaSize(rawSmSize-extSmSize) - (rawSmSize - extSmSize))
			// It is possible that for the bytes scenario, the extracted field contains no scope metrics.
			// Do not add it to the destination if that is the case.
			if extSrcSM.Metrics().Len() > 0 {
				extSrcSM.MoveTo(destRM.ScopeMetrics().AppendEmpty())
			}
			return extSrcSM.Metrics().Len() != 0
		}
		capacityLeft -= smSize
		removedSize += smSize
		srcSM.MoveTo(destRM.ScopeMetrics().AppendEmpty())
		return true
	})
	return destRM, removedSize
}

// extractScopeMetrics extracts scope metrics and returns a new scope metrics with the specified number of data points.
func extractScopeMetrics(srcSM pmetric.ScopeMetrics, capacity int, sz sizer.MetricsSizer) (pmetric.ScopeMetrics, int) {
	destSM := pmetric.NewScopeMetrics()
	destSM.SetSchemaUrl(srcSM.SchemaUrl())
	srcSM.Scope().CopyTo(destSM.Scope())
	// Take into account that this can have max "capacity", so when added to the parent will need space for the extra delta size.
	capacityLeft := capacity - (sz.DeltaSize(capacity) - capacity) - sz.ScopeMetricsSize(destSM)
	removedSize := 0
	srcSM.Metrics().RemoveIf(func(srcSM pmetric.Metric) bool {
		// If the no more capacity left just return.
		if capacityLeft == 0 {
			return false
		}
		rawRmSize := sz.MetricSize(srcSM)
		rmSize := sz.DeltaSize(rawRmSize)
		if rmSize > capacityLeft {
			extSrcDP, extRmSize := extractMetricDataPoints(srcSM, capacityLeft, sz)
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			removedSize += extRmSize
			// There represents the delta between the delta sizes.
			removedSize += rmSize - rawRmSize - (sz.DeltaSize(rawRmSize-extRmSize) - (rawRmSize - extRmSize))
			// It is possible that for the bytes scenario, the extracted field contains no datapoints.
			// Do not add it to the destination if that is the case.
			if dataPointsLen(extSrcDP) > 0 {
				extSrcDP.MoveTo(destSM.Metrics().AppendEmpty())
			}
			return dataPointsLen(extSrcDP) != 0
		}
		capacityLeft -= rmSize
		removedSize += rmSize
		srcSM.MoveTo(destSM.Metrics().AppendEmpty())
		return true
	})
	return destSM, removedSize
}

func extractMetricDataPoints(srcMetric pmetric.Metric, capacity int, sz sizer.MetricsSizer) (pmetric.Metric, int) {
	destMetric := pmetric.NewMetric()
	destMetric.SetName(srcMetric.Name())
	destMetric.SetDescription(srcMetric.Description())
	destMetric.SetUnit(srcMetric.Unit())
	srcMetric.Metadata().CopyTo(destMetric.Metadata())

	var removedSize int
	switch srcMetric.Type() {
	case pmetric.MetricTypeGauge:
		removedSize = extractGaugeDataPoints(srcMetric.Gauge(), destMetric, capacity, sz)
	case pmetric.MetricTypeSum:
		removedSize = extractSumDataPoints(srcMetric.Sum(), destMetric, capacity, sz)
		destMetric.Sum().SetIsMonotonic(srcMetric.Sum().IsMonotonic())
		destMetric.Sum().SetAggregationTemporality(srcMetric.Sum().AggregationTemporality())
	case pmetric.MetricTypeHistogram:
		removedSize = extractHistogramDataPoints(srcMetric.Histogram(), destMetric, capacity, sz)
		destMetric.Histogram().SetAggregationTemporality(srcMetric.Histogram().AggregationTemporality())
	case pmetric.MetricTypeExponentialHistogram:
		removedSize = extractExponentialHistogramDataPoints(srcMetric.ExponentialHistogram(), destMetric, capacity, sz)
		destMetric.ExponentialHistogram().SetAggregationTemporality(srcMetric.ExponentialHistogram().AggregationTemporality())
	case pmetric.MetricTypeSummary:
		removedSize = extractSummaryDataPoints(srcMetric.Summary(), destMetric, capacity, sz)
	}
	return destMetric, removedSize
}

func dataPointsLen(m pmetric.Metric) int {
	switch m.Type() {
	case pmetric.MetricTypeGauge:
		return m.Gauge().DataPoints().Len()
	case pmetric.MetricTypeSum:
		return m.Sum().DataPoints().Len()
	case pmetric.MetricTypeHistogram:
		return m.Histogram().DataPoints().Len()
	case pmetric.MetricTypeExponentialHistogram:
		return m.ExponentialHistogram().DataPoints().Len()
	case pmetric.MetricTypeSummary:
		return m.Summary().DataPoints().Len()
	}
	return 0
}

func extractGaugeDataPoints(srcGauge pmetric.Gauge, destMetric pmetric.Metric, capacity int, sz sizer.MetricsSizer) int {
	destGauge := destMetric.SetEmptyGauge()

	// Take into account that this can have max "capacity", so when added to the parent will need space for the extra delta size.
	capacityLeft := capacity - (sz.DeltaSize(capacity) - capacity) - sz.MetricSize(destMetric)
	removedSize := 0

	srcGauge.DataPoints().RemoveIf(func(srcDP pmetric.NumberDataPoint) bool {
		// If the no more capacity left just return.
		if capacityLeft == 0 {
			return false
		}

		rdSize := sz.DeltaSize(sz.NumberDataPointSize(srcDP))
		if rdSize > capacityLeft {
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			return false
		}
		capacityLeft -= rdSize
		removedSize += rdSize
		srcDP.MoveTo(destGauge.DataPoints().AppendEmpty())
		return true
	})
	return removedSize
}

func extractSumDataPoints(srcSum pmetric.Sum, destMetric pmetric.Metric, capacity int, sz sizer.MetricsSizer) int {
	destSum := destMetric.SetEmptySum()
	// Take into account that this can have max "capacity", so when added to the parent will need space for the extra delta size.
	capacityLeft := capacity - (sz.DeltaSize(capacity) - capacity) - sz.MetricSize(destMetric)
	removedSize := 0
	srcSum.DataPoints().RemoveIf(func(srcDP pmetric.NumberDataPoint) bool {
		// If the no more capacity left just return.
		if capacityLeft == 0 {
			return false
		}

		rdSize := sz.DeltaSize(sz.NumberDataPointSize(srcDP))
		if rdSize > capacityLeft {
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			return false
		}
		capacityLeft -= rdSize
		removedSize += rdSize
		srcDP.MoveTo(destSum.DataPoints().AppendEmpty())
		return true
	})
	return removedSize
}

func extractHistogramDataPoints(srcHistogram pmetric.Histogram, destMetric pmetric.Metric, capacity int, sz sizer.MetricsSizer) int {
	destHistogram := destMetric.SetEmptyHistogram()
	// Take into account that this can have max "capacity", so when added to the parent will need space for the extra delta size.
	capacityLeft := capacity - (sz.DeltaSize(capacity) - capacity) - sz.MetricSize(destMetric)
	removedSize := 0
	srcHistogram.DataPoints().RemoveIf(func(srcDP pmetric.HistogramDataPoint) bool {
		// If the no more capacity left just return.
		if capacityLeft == 0 {
			return false
		}

		rdSize := sz.DeltaSize(sz.HistogramDataPointSize(srcDP))
		if rdSize > capacityLeft {
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			return false
		}
		capacityLeft -= rdSize
		removedSize += rdSize
		srcDP.MoveTo(destHistogram.DataPoints().AppendEmpty())
		return true
	})
	return removedSize
}

func extractExponentialHistogramDataPoints(srcExponentialHistogram pmetric.ExponentialHistogram, destMetric pmetric.Metric, capacity int, sz sizer.MetricsSizer) int {
	destExponentialHistogram := destMetric.SetEmptyExponentialHistogram()
	// Take into account that this can have max "capacity", so when added to the parent will need space for the extra delta size.
	capacityLeft := capacity - (sz.DeltaSize(capacity) - capacity) - sz.MetricSize(destMetric)
	removedSize := 0
	srcExponentialHistogram.DataPoints().RemoveIf(func(srcDP pmetric.ExponentialHistogramDataPoint) bool {
		// If the no more capacity left just return.
		if capacityLeft == 0 {
			return false
		}

		rdSize := sz.DeltaSize(sz.ExponentialHistogramDataPointSize(srcDP))
		if rdSize > capacityLeft {
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			return false
		}
		capacityLeft -= rdSize
		removedSize += rdSize
		srcDP.MoveTo(destExponentialHistogram.DataPoints().AppendEmpty())
		return true
	})
	return removedSize
}

func extractSummaryDataPoints(srcSummary pmetric.Summary, destMetric pmetric.Metric, capacity int, sz sizer.MetricsSizer) int {
	destSummary := destMetric.SetEmptySummary()
	// Take into account that this can have max "capacity", so when added to the parent will need space for the extra delta size.
	capacityLeft := capacity - (sz.DeltaSize(capacity) - capacity) - sz.MetricSize(destMetric)
	removedSize := 0
	srcSummary.DataPoints().RemoveIf(func(srcDP pmetric.SummaryDataPoint) bool {
		// If the no more capacity left just return.
		if capacityLeft == 0 {
			return false
		}

		rdSize := sz.DeltaSize(sz.SummaryDataPointSize(srcDP))
		if rdSize > capacityLeft {
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			return false
		}
		capacityLeft -= rdSize
		removedSize += rdSize
		srcDP.MoveTo(destSummary.DataPoints().AppendEmpty())
		return true
	})
	return removedSize
}
