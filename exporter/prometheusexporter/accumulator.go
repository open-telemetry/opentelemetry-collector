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
	// value contains a metric with exactly one aggregated datapoint
	value pdata.Metric
	// stored indicates when metric was stored
	stored time.Time

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
	// should be served after it was stored
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
	ilms := rm.InstrumentationLibraryMetrics()

	for i := 0; i < ilms.Len(); i++ {
		ilm := ilms.At(i)

		metrics := ilm.Metrics()
		for j := 0; j < metrics.Len(); j++ {
			n += a.addMetric(metrics.At(j), ilm.InstrumentationLibrary())
		}
	}

	return
}

func (a *lastValueAccumulator) addMetric(metric pdata.Metric, il pdata.InstrumentationLibrary) int {
	a.logger.Debug(fmt.Sprintf("accumulating metric: %s", metric.Name()))

	switch metric.DataType() {
	case pdata.MetricDataTypeIntGauge:
		return a.accumulateIntGauge(metric, il)
	case pdata.MetricDataTypeIntSum:
		return a.accumulateIntSum(metric, il)
	case pdata.MetricDataTypeDoubleGauge:
		return a.accumulateDoubleGauge(metric, il)
	case pdata.MetricDataTypeDoubleSum:
		return a.accumulateDoubleSum(metric, il)
	case pdata.MetricDataTypeIntHistogram:
		return a.accumulateIntHistogram(metric, il)
	case pdata.MetricDataTypeDoubleHistogram:
		return a.accumulateDoubleHistogram(metric, il)
	}

	return 0
}

func (a *lastValueAccumulator) accumulateIntGauge(metric pdata.Metric, il pdata.InstrumentationLibrary) (n int) {
	dps := metric.IntGauge().DataPoints()
	for i := 0; i < dps.Len(); i++ {
		ip := dps.At(i)

		ts := ip.Timestamp().AsTime()
		signature := timeseriesSignature(il.Name(), metric, ip.LabelsMap())

		v, ok := a.registeredMetrics.Load(signature)
		if !ok {
			m := createMetric(metric)
			m.IntGauge().DataPoints().Append(ip)
			a.registeredMetrics.Store(signature, &accumulatedValue{value: m, instrumentationLibrary: il, stored: time.Now()})
			n++
			continue
		}
		mv := v.(*accumulatedValue)

		if ts.Before(mv.value.IntGauge().DataPoints().At(0).Timestamp().AsTime()) {
			// only keep datapoint with latest timestamp
			continue
		}

		m := createMetric(metric)
		m.IntGauge().DataPoints().Append(ip)
		a.registeredMetrics.Store(signature, &accumulatedValue{value: m, instrumentationLibrary: il, stored: time.Now()})
		n++
	}
	return
}

func (a *lastValueAccumulator) accumulateDoubleGauge(metric pdata.Metric, il pdata.InstrumentationLibrary) (n int) {
	dps := metric.DoubleGauge().DataPoints()
	for i := 0; i < dps.Len(); i++ {
		ip := dps.At(i)

		ts := ip.Timestamp().AsTime()
		signature := timeseriesSignature(il.Name(), metric, ip.LabelsMap())

		v, ok := a.registeredMetrics.Load(signature)
		if !ok {
			m := createMetric(metric)
			m.DoubleGauge().DataPoints().Append(ip)
			a.registeredMetrics.Store(signature, &accumulatedValue{value: m, instrumentationLibrary: il, stored: time.Now()})
			n++
			continue
		}
		mv := v.(*accumulatedValue)

		if ts.Before(mv.value.DoubleGauge().DataPoints().At(0).Timestamp().AsTime()) {
			// only keep datapoint with latest timestamp
			continue
		}

		m := createMetric(metric)
		m.DoubleGauge().DataPoints().Append(ip)
		a.registeredMetrics.Store(signature, &accumulatedValue{value: m, instrumentationLibrary: il, stored: time.Now()})
		n++
	}
	return
}

func (a *lastValueAccumulator) accumulateIntSum(metric pdata.Metric, il pdata.InstrumentationLibrary) (n int) {
	intSum := metric.IntSum()

	// Drop metrics with non-cumulative aggregations
	if intSum.AggregationTemporality() != pdata.AggregationTemporalityCumulative {
		return
	}

	dps := intSum.DataPoints()
	for i := 0; i < dps.Len(); i++ {
		ip := dps.At(i)

		ts := ip.Timestamp().AsTime()
		signature := timeseriesSignature(il.Name(), metric, ip.LabelsMap())

		v, ok := a.registeredMetrics.Load(signature)
		if !ok {
			m := createMetric(metric)
			m.IntSum().SetIsMonotonic(metric.IntSum().IsMonotonic())
			m.IntSum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
			m.IntSum().DataPoints().Append(ip)
			a.registeredMetrics.Store(signature, &accumulatedValue{value: m, instrumentationLibrary: il, stored: time.Now()})
			n++
			continue
		}
		mv := v.(*accumulatedValue)

		if ts.Before(mv.value.IntSum().DataPoints().At(0).Timestamp().AsTime()) {
			// only keep datapoint with latest timestamp
			continue
		}

		m := createMetric(metric)
		m.IntSum().SetIsMonotonic(metric.IntSum().IsMonotonic())
		m.IntSum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
		m.IntSum().DataPoints().Append(ip)
		a.registeredMetrics.Store(signature, &accumulatedValue{value: m, instrumentationLibrary: il, stored: time.Now()})
		n++
	}
	return
}

func (a *lastValueAccumulator) accumulateDoubleSum(metric pdata.Metric, il pdata.InstrumentationLibrary) (n int) {
	doubleSum := metric.DoubleSum()

	// Drop metrics with non-cumulative aggregations
	if doubleSum.AggregationTemporality() != pdata.AggregationTemporalityCumulative {
		return
	}

	dps := doubleSum.DataPoints()
	for i := 0; i < dps.Len(); i++ {
		ip := dps.At(i)

		ts := ip.Timestamp().AsTime()
		signature := timeseriesSignature(il.Name(), metric, ip.LabelsMap())

		v, ok := a.registeredMetrics.Load(signature)
		if !ok {
			m := createMetric(metric)
			m.DoubleSum().SetIsMonotonic(metric.DoubleSum().IsMonotonic())
			m.DoubleSum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
			m.DoubleSum().DataPoints().Append(ip)
			a.registeredMetrics.Store(signature, &accumulatedValue{value: m, instrumentationLibrary: il, stored: time.Now()})
			n++
			continue
		}
		mv := v.(*accumulatedValue)

		if ts.Before(mv.value.DoubleSum().DataPoints().At(0).Timestamp().AsTime()) {
			// only keep datapoint with latest timestamp
			continue
		}

		m := createMetric(metric)
		m.DoubleSum().SetIsMonotonic(metric.DoubleSum().IsMonotonic())
		m.DoubleSum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
		m.DoubleSum().DataPoints().Append(ip)
		a.registeredMetrics.Store(signature, &accumulatedValue{value: m, instrumentationLibrary: il, stored: time.Now()})
		n++
	}
	return
}

func (a *lastValueAccumulator) accumulateIntHistogram(metric pdata.Metric, il pdata.InstrumentationLibrary) (n int) {
	intHistogram := metric.IntHistogram()

	// Drop metrics with non-cumulative aggregations
	if intHistogram.AggregationTemporality() != pdata.AggregationTemporalityCumulative {
		return
	}

	dps := intHistogram.DataPoints()
	for i := 0; i < dps.Len(); i++ {
		ip := dps.At(i)

		ts := ip.Timestamp().AsTime()
		signature := timeseriesSignature(il.Name(), metric, ip.LabelsMap())

		v, ok := a.registeredMetrics.Load(signature)
		if !ok {
			m := createMetric(metric)
			m.IntHistogram().DataPoints().Append(ip)
			a.registeredMetrics.Store(signature, &accumulatedValue{value: m, instrumentationLibrary: il, stored: time.Now()})
			n++
			continue
		}
		mv := v.(*accumulatedValue)

		if ts.Before(mv.value.IntHistogram().DataPoints().At(0).Timestamp().AsTime()) {
			// only keep datapoint with latest timestamp
			continue
		}

		m := createMetric(metric)
		m.IntHistogram().DataPoints().Append(ip)
		m.IntHistogram().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
		a.registeredMetrics.Store(signature, &accumulatedValue{value: m, instrumentationLibrary: il, stored: time.Now()})
		n++
	}
	return
}

func (a *lastValueAccumulator) accumulateDoubleHistogram(metric pdata.Metric, il pdata.InstrumentationLibrary) (n int) {
	doubleHistogram := metric.DoubleHistogram()

	// Drop metrics with non-cumulative aggregations
	if doubleHistogram.AggregationTemporality() != pdata.AggregationTemporalityCumulative {
		return
	}

	dps := doubleHistogram.DataPoints()
	for i := 0; i < dps.Len(); i++ {
		ip := dps.At(i)

		ts := ip.Timestamp().AsTime()
		signature := timeseriesSignature(il.Name(), metric, ip.LabelsMap())

		v, ok := a.registeredMetrics.Load(signature)
		if !ok {
			m := createMetric(metric)
			m.DoubleHistogram().DataPoints().Append(ip)
			a.registeredMetrics.Store(signature, &accumulatedValue{value: m, instrumentationLibrary: il, stored: time.Now()})
			n++
			continue
		}
		mv := v.(*accumulatedValue)

		if ts.Before(mv.value.DoubleHistogram().DataPoints().At(0).Timestamp().AsTime()) {
			// only keep datapoint with latest timestamp
			continue
		}

		m := createMetric(metric)
		m.DoubleHistogram().DataPoints().Append(ip)
		m.DoubleHistogram().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
		a.registeredMetrics.Store(signature, &accumulatedValue{value: m, instrumentationLibrary: il, stored: time.Now()})
		n++
	}
	return
}

// Collect returns a slice with relevant aggregated metrics
func (a *lastValueAccumulator) Collect() []pdata.Metric {
	a.logger.Debug("Accumulator collect called")

	res := make([]pdata.Metric, 0)

	a.registeredMetrics.Range(func(key, value interface{}) bool {
		v := value.(*accumulatedValue)
		if time.Now().After(v.stored.Add(a.metricExpiration)) {
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
	labels.ForEach(func(k string, v string) {
		b.WriteString("*" + k + "*" + v)
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
