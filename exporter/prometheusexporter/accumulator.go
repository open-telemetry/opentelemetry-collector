// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package prometheusexporter

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/consumer/pdata"
)

type accumulatedValue struct {
	// value contains a metric with exactly one aggregated datapoint.
	value pdata.Metric
	// updated indicates when metric was last changed.
	updated time.Time

	instrumentationLibrary pdata.InstrumentationLibrary
}

// accumulator stores aggragated values of incoming metrics
type accumulator interface {
	// Accumulate stores aggragated metric values
	Accumulate(resourceMetrics pdata.ResourceMetrics) (processed int)
	// Collect returns a slice with relevant aggregated metrics
	Collect() (metrics []pdata.Metric)
}

// LastValueAccumulator keeps last value for accumulated metrics
type lastValueAccumulator struct {
	logger *zap.Logger

	registeredMetrics sync.Map

	// metricExpiration contains duration for which metric
	// should be served after it was updated
	metricExpiration time.Duration
}

// NewAccumulator returns LastValueAccumulator
func newAccumulator(logger *zap.Logger, metricExpiration time.Duration) accumulator {
	return &lastValueAccumulator{
		logger:           logger,
		metricExpiration: metricExpiration,
	}
}

// Accumulate stores one datapoint per metric
func (a *lastValueAccumulator) Accumulate(rm pdata.ResourceMetrics) (n int) {
	now := time.Now()
	ilms := rm.InstrumentationLibraryMetrics()

	for i := 0; i < ilms.Len(); i++ {
		ilm := ilms.At(i)

		metrics := ilm.Metrics()
		for j := 0; j < metrics.Len(); j++ {
			n += a.addMetric(metrics.At(j), ilm.InstrumentationLibrary(), now)
		}
	}

	return
}

func (a *lastValueAccumulator) addMetric(metric pdata.Metric, il pdata.InstrumentationLibrary, now time.Time) int {
	a.logger.Debug(fmt.Sprintf("accumulating metric: %s", metric.Name()))

	switch metric.DataType() {
	case pdata.MetricDataTypeIntGauge:
		return a.accumulateIntGauge(metric, il, now)
	case pdata.MetricDataTypeIntSum:
		return a.accumulateIntSum(metric, il, now)
	case pdata.MetricDataTypeDoubleGauge:
		return a.accumulateDoubleGauge(metric, il, now)
	case pdata.MetricDataTypeDoubleSum:
		return a.accumulateDoubleSum(metric, il, now)
	case pdata.MetricDataTypeIntHistogram:
		return a.accumulateIntHistogram(metric, il, now)
	case pdata.MetricDataTypeHistogram:
		return a.accumulateDoubleHistogram(metric, il, now)
	case pdata.MetricDataTypeSummary:
		return a.accumulateSummary(metric, il, now)
	default:
		a.logger.With(
			zap.String("data_type", string(metric.DataType())),
			zap.String("metric_name", metric.Name()),
		).Error("failed to translate metric")
	}

	return 0
}

func (a *lastValueAccumulator) accumulateSummary(metric pdata.Metric, il pdata.InstrumentationLibrary, now time.Time) (n int) {
	dps := metric.Summary().DataPoints()
	for i := 0; i < dps.Len(); i++ {
		ip := dps.At(i)

		signature := timeseriesSignature(il.Name(), metric, ip.LabelsMap())

		v, ok := a.registeredMetrics.Load(signature)
		stalePoint := ok &&
			ip.Timestamp().AsTime().Before(v.(*accumulatedValue).value.Summary().DataPoints().At(0).Timestamp().AsTime())

		if stalePoint {
			// Only keep this datapoint if it has a later timestamp.
			continue
		}

		mm := createMetric(metric)
		ip.CopyTo(mm.Summary().DataPoints().AppendEmpty())
		a.registeredMetrics.Store(signature, &accumulatedValue{value: mm, instrumentationLibrary: il, updated: now})
		n++
	}

	return n
}

func (a *lastValueAccumulator) accumulateIntGauge(metric pdata.Metric, il pdata.InstrumentationLibrary, now time.Time) (n int) {
	dps := metric.IntGauge().DataPoints()
	for i := 0; i < dps.Len(); i++ {
		ip := dps.At(i)

		signature := timeseriesSignature(il.Name(), metric, ip.LabelsMap())

		v, ok := a.registeredMetrics.Load(signature)
		if !ok {
			m := createMetric(metric)
			ip.CopyTo(m.IntGauge().DataPoints().AppendEmpty())
			a.registeredMetrics.Store(signature, &accumulatedValue{value: m, instrumentationLibrary: il, updated: now})
			n++
			continue
		}
		mv := v.(*accumulatedValue)

		if ip.Timestamp().AsTime().Before(mv.value.IntGauge().DataPoints().At(0).Timestamp().AsTime()) {
			// only keep datapoint with latest timestamp
			continue
		}

		m := createMetric(metric)
		ip.CopyTo(m.IntGauge().DataPoints().AppendEmpty())
		a.registeredMetrics.Store(signature, &accumulatedValue{value: m, instrumentationLibrary: il, updated: now})
		n++
	}
	return
}

func (a *lastValueAccumulator) accumulateDoubleGauge(metric pdata.Metric, il pdata.InstrumentationLibrary, now time.Time) (n int) {
	dps := metric.DoubleGauge().DataPoints()
	for i := 0; i < dps.Len(); i++ {
		ip := dps.At(i)

		signature := timeseriesSignature(il.Name(), metric, ip.LabelsMap())

		v, ok := a.registeredMetrics.Load(signature)
		if !ok {
			m := createMetric(metric)
			ip.CopyTo(m.DoubleGauge().DataPoints().AppendEmpty())
			a.registeredMetrics.Store(signature, &accumulatedValue{value: m, instrumentationLibrary: il, updated: now})
			n++
			continue
		}
		mv := v.(*accumulatedValue)

		if ip.Timestamp().AsTime().Before(mv.value.DoubleGauge().DataPoints().At(0).Timestamp().AsTime()) {
			// only keep datapoint with latest timestamp
			continue
		}

		m := createMetric(metric)
		ip.CopyTo(m.DoubleGauge().DataPoints().AppendEmpty())
		a.registeredMetrics.Store(signature, &accumulatedValue{value: m, instrumentationLibrary: il, updated: now})
		n++
	}
	return
}

func (a *lastValueAccumulator) accumulateIntSum(metric pdata.Metric, il pdata.InstrumentationLibrary, now time.Time) (n int) {
	intSum := metric.IntSum()

	// Drop metrics with non-cumulative aggregations
	if intSum.AggregationTemporality() != pdata.AggregationTemporalityCumulative {
		return
	}

	dps := intSum.DataPoints()
	for i := 0; i < dps.Len(); i++ {
		ip := dps.At(i)

		signature := timeseriesSignature(il.Name(), metric, ip.LabelsMap())

		v, ok := a.registeredMetrics.Load(signature)
		if !ok {
			m := createMetric(metric)
			m.IntSum().SetIsMonotonic(metric.IntSum().IsMonotonic())
			m.IntSum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
			ip.CopyTo(m.IntSum().DataPoints().AppendEmpty())
			a.registeredMetrics.Store(signature, &accumulatedValue{value: m, instrumentationLibrary: il, updated: now})
			n++
			continue
		}
		mv := v.(*accumulatedValue)

		if ip.Timestamp().AsTime().Before(mv.value.IntSum().DataPoints().At(0).Timestamp().AsTime()) {
			// only keep datapoint with latest timestamp
			continue
		}

		m := createMetric(metric)
		m.IntSum().SetIsMonotonic(metric.IntSum().IsMonotonic())
		m.IntSum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
		ip.CopyTo(m.IntSum().DataPoints().AppendEmpty())
		a.registeredMetrics.Store(signature, &accumulatedValue{value: m, instrumentationLibrary: il, updated: now})
		n++
	}
	return
}

func (a *lastValueAccumulator) accumulateDoubleSum(metric pdata.Metric, il pdata.InstrumentationLibrary, now time.Time) (n int) {
	doubleSum := metric.DoubleSum()

	// Drop metrics with non-cumulative aggregations
	if doubleSum.AggregationTemporality() != pdata.AggregationTemporalityCumulative {
		return
	}

	dps := doubleSum.DataPoints()
	for i := 0; i < dps.Len(); i++ {
		ip := dps.At(i)

		signature := timeseriesSignature(il.Name(), metric, ip.LabelsMap())

		v, ok := a.registeredMetrics.Load(signature)
		if !ok {
			m := createMetric(metric)
			m.DoubleSum().SetIsMonotonic(metric.DoubleSum().IsMonotonic())
			m.DoubleSum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
			ip.CopyTo(m.DoubleSum().DataPoints().AppendEmpty())
			a.registeredMetrics.Store(signature, &accumulatedValue{value: m, instrumentationLibrary: il, updated: now})
			n++
			continue
		}
		mv := v.(*accumulatedValue)

		if ip.Timestamp().AsTime().Before(mv.value.DoubleSum().DataPoints().At(0).Timestamp().AsTime()) {
			// only keep datapoint with latest timestamp
			continue
		}

		m := createMetric(metric)
		m.DoubleSum().SetIsMonotonic(metric.DoubleSum().IsMonotonic())
		m.DoubleSum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
		ip.CopyTo(m.DoubleSum().DataPoints().AppendEmpty())
		a.registeredMetrics.Store(signature, &accumulatedValue{value: m, instrumentationLibrary: il, updated: now})
		n++
	}
	return
}

func (a *lastValueAccumulator) accumulateIntHistogram(metric pdata.Metric, il pdata.InstrumentationLibrary, now time.Time) (n int) {
	intHistogram := metric.IntHistogram()

	// Drop metrics with non-cumulative aggregations
	if intHistogram.AggregationTemporality() != pdata.AggregationTemporalityCumulative {
		return
	}

	dps := intHistogram.DataPoints()
	for i := 0; i < dps.Len(); i++ {
		ip := dps.At(i)

		signature := timeseriesSignature(il.Name(), metric, ip.LabelsMap())

		v, ok := a.registeredMetrics.Load(signature)
		if !ok {
			m := createMetric(metric)
			ip.CopyTo(m.IntHistogram().DataPoints().AppendEmpty())
			a.registeredMetrics.Store(signature, &accumulatedValue{value: m, instrumentationLibrary: il, updated: now})
			n++
			continue
		}
		mv := v.(*accumulatedValue)

		if ip.Timestamp().AsTime().Before(mv.value.IntHistogram().DataPoints().At(0).Timestamp().AsTime()) {
			// only keep datapoint with latest timestamp
			continue
		}

		m := createMetric(metric)
		ip.CopyTo(m.IntHistogram().DataPoints().AppendEmpty())
		m.IntHistogram().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
		a.registeredMetrics.Store(signature, &accumulatedValue{value: m, instrumentationLibrary: il, updated: now})
		n++
	}
	return
}

func (a *lastValueAccumulator) accumulateDoubleHistogram(metric pdata.Metric, il pdata.InstrumentationLibrary, now time.Time) (n int) {
	doubleHistogram := metric.Histogram()

	// Drop metrics with non-cumulative aggregations
	if doubleHistogram.AggregationTemporality() != pdata.AggregationTemporalityCumulative {
		return
	}

	dps := doubleHistogram.DataPoints()
	for i := 0; i < dps.Len(); i++ {
		ip := dps.At(i)

		signature := timeseriesSignature(il.Name(), metric, ip.LabelsMap())

		v, ok := a.registeredMetrics.Load(signature)
		if !ok {
			m := createMetric(metric)
			ip.CopyTo(m.Histogram().DataPoints().AppendEmpty())
			a.registeredMetrics.Store(signature, &accumulatedValue{value: m, instrumentationLibrary: il, updated: now})
			n++
			continue
		}
		mv := v.(*accumulatedValue)

		if ip.Timestamp().AsTime().Before(mv.value.Histogram().DataPoints().At(0).Timestamp().AsTime()) {
			// only keep datapoint with latest timestamp
			continue
		}

		m := createMetric(metric)
		ip.CopyTo(m.Histogram().DataPoints().AppendEmpty())
		m.Histogram().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
		a.registeredMetrics.Store(signature, &accumulatedValue{value: m, instrumentationLibrary: il, updated: now})
		n++
	}
	return
}

// Collect returns a slice with relevant aggregated metrics
func (a *lastValueAccumulator) Collect() []pdata.Metric {
	a.logger.Debug("Accumulator collect called")

	var res []pdata.Metric
	expirationTime := time.Now().Add(-a.metricExpiration)

	a.registeredMetrics.Range(func(key, value interface{}) bool {
		v := value.(*accumulatedValue)
		if expirationTime.After(v.updated) {
			a.logger.Debug(fmt.Sprintf("metric expired: %s", v.value.Name()))
			a.registeredMetrics.Delete(key)
			return true
		}

		res = append(res, v.value)
		return true
	})

	return res
}

func timeseriesSignature(ilmName string, metric pdata.Metric, labels pdata.StringMap) string {
	var b strings.Builder
	b.WriteString(metric.DataType().String())
	b.WriteString("*" + ilmName)
	b.WriteString("*" + metric.Name())
	labels.Sort().Range(func(k string, v string) bool {
		b.WriteString("*" + k + "*" + v)
		return true
	})
	return b.String()
}

func createMetric(metric pdata.Metric) pdata.Metric {
	m := pdata.NewMetric()
	m.SetName(metric.Name())
	m.SetDescription(metric.Description())
	m.SetUnit(metric.Unit())
	m.SetDataType(metric.DataType())

	return m
}
