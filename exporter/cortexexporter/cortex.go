// Copyright 2020 The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cortexexporter

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"log"
	"github.com/prometheus/prometheus/prompb"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/consumer/pdatautil"
	"go.opentelemetry.io/collector/internal/data"
	common "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/common/v1"
	otlp "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/metrics/v1"

)
// TODO: get default labels such as job or instance from Resource
type cortexExporter struct {
	namespace 	string
	endpoint  	string
	client    	*http.Client
	headers		map[string]string
	wg		  	*sync.WaitGroup
	closeChan	chan struct{}
}

// ByLabelName enables the usage of sort.Sort() with a slice of labels
type ByLabelName []prompb.Label
func (a ByLabelName) Len() int           { return len(a) }
func (a ByLabelName) Less(i, j int) bool { return a[i].Name < a[j].Name }
func (a ByLabelName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }


// validateMetrics returns a bool representing whether the metric has a valid type and temporality combination.
func validateMetrics(desc *otlp.MetricDescriptor) bool {
	if desc == nil {
		return false
	}
	switch desc.GetType() {
		case otlp.MetricDescriptor_MONOTONIC_DOUBLE, otlp.MetricDescriptor_MONOTONIC_INT64,
			otlp.MetricDescriptor_HISTOGRAM, otlp.MetricDescriptor_SUMMARY:
			return desc.GetTemporality() == otlp.MetricDescriptor_CUMULATIVE
		case otlp.MetricDescriptor_INT64, otlp.MetricDescriptor_DOUBLE:
			return true
	}
	return false
}

// addSample finds a TimeSeries in tsMap that corresponds to the label set lbs, and add sample to the TimeSeries; it
// creates a new TimeSeries in the map if not found. tsMap is unmodified if either of its parameters is nil.
func addSample(tsMap map[string]*prompb.TimeSeries, sample *prompb.Sample, lbs []prompb.Label,
	ty otlp.MetricDescriptor_Type) {
	if sample == nil || lbs == nil || tsMap == nil {
		return
	}
	sig := timeSeriesSignature(ty, &lbs)
	ts, ok := tsMap[sig]
	if ok {
		ts.Samples = append(ts.Samples, *sample)
	} else {
		newTs := &prompb.TimeSeries{
			Labels:		lbs,
			Samples:	[]prompb.Sample{*sample},
		}
		tsMap[sig] = newTs
	}
}

// timeSeries return a string signature in the form of:
// 		TYPE-label1-value1- ...  -labelN-valueN
// the label slice should not contain duplicate label names; this method sorts the slice by label name before creating
// the signature.
func timeSeriesSignature(t otlp.MetricDescriptor_Type, lbs *[]prompb.Label) string {
	b := strings.Builder{}
	fmt.Fprintf(&b, t.String())

	sort.Sort(ByLabelName(*lbs))

	for _, lb := range *lbs {
		fmt.Fprintf(&b, "-%s-%s", lb.GetName(),lb.GetValue())
	}

	return b.String()
}

// createLabelSet creates a slice of Cortex Label with OTLP labels and paris of string values.
// Unpaired string value is ignored. String pairs overwrites OTLP labels if collision happens, and the overwrite is
// logged. Resultant label names are sanitized.
func createLabelSet(labels []*common.StringKeyValue, extras ...string) []prompb.Label {
	l := map[string]prompb.Label{}
	for _, lb := range labels {
		l[lb.Key] = prompb.Label{
			Name:	sanitize(lb.Key),
			Value:	lb.Value,
		}
	}
	for i := 0; i < len(extras); i+=2 {
		if i+1 >= len(extras){
			break
		}
		_,found:= l[extras[i]]
		if found {
			log.Println("label " + extras[i] + " is overwritten. Check if Prometheus reserved labels are used.")
		}
 		l[extras[i]] = prompb.Label{
			Name:   sanitize(extras[i]),
			Value:	extras[i+1],
		}
	}
	s := make([]prompb.Label,0,len(l))
	for _, lb := range l {
		s = append(s,lb)
	}
	return s
}

// handleScalarMetric processes data points in a single OTLP scalar metric by adding the each point as a Sample into
// its corresponding TimeSeries in tsMap. tsMap and metric cannot be nil. metric must have a non-nil descriptor
func (ce *cortexExporter) handleScalarMetric(tsMap map[string]*prompb.TimeSeries, metric *otlp.Metric) error {

	ty := metric.MetricDescriptor.Type
	switch ty {
	// int points
	case otlp.MetricDescriptor_MONOTONIC_INT64,otlp.MetricDescriptor_INT64:
		if metric.Int64DataPoints == nil {
			return fmt.Errorf("nil data point field in metric" + metric.GetMetricDescriptor().Name)
		}
		for _, pt := range metric.Int64DataPoints {
			name := getPromMetricName(metric.GetMetricDescriptor(), ce.namespace)
			lbs := createLabelSet(pt.GetLabels(), "name", name)
			sample := &prompb.Sample{
				Value: 		float64(pt.Value),
				Timestamp: 	int64(pt.TimeUnixNano),
			}
			addSample(tsMap,sample, lbs, metric.GetMetricDescriptor().GetType())
		}
		return nil

	// double points
	case otlp.MetricDescriptor_MONOTONIC_DOUBLE, otlp.MetricDescriptor_DOUBLE:
		if metric.DoubleDataPoints == nil {
			return fmt.Errorf("nil data point field in metric" + metric.GetMetricDescriptor().Name)
		}
		for _, pt := range metric.DoubleDataPoints {
			name := getPromMetricName(metric.GetMetricDescriptor(), ce.namespace)
			lbs := createLabelSet(pt.GetLabels(),"name", name)
			sample := &prompb.Sample{
				Value: 		pt.Value,
				Timestamp: 	int64(pt.TimeUnixNano),
			}
			addSample(tsMap,sample, lbs, metric.GetMetricDescriptor().GetType())
		}
		return nil
	}

	return fmt.Errorf("invalid metric type: wants int or double data points");
}

// handleHistogramMetric processes data points in a single OTLP histogram metric by mapping the sum, count and each
// bucket of every data point as a Sample, and adding each Sample to its corresponding TimeSeries.
// tsMap and metric cannot be nil.
func (ce *cortexExporter) handleHistogramMetric(tsMap map[string]*prompb.TimeSeries, metric *otlp.Metric) error {
	if metric.HistogramDataPoints == nil {
		return fmt.Errorf("invalid metric type: wants histogram points")
	}
	for _, pt := range metric.HistogramDataPoints {
		time := int64(pt.GetTimeUnixNano())
		ty := metric.GetMetricDescriptor().GetType()
		baseName := getPromMetricName(metric.GetMetricDescriptor(),ce.namespace)
		sum := &prompb.Sample{
			Value:		pt.GetSum(),
			Timestamp:	time,
		}
		count := &prompb.Sample{
			Value:		float64(pt.GetCount()),
			Timestamp:	time,
		}
		sumLbs := createLabelSet(pt.GetLabels(),"name", baseName+"_sum")
		countLbs := createLabelSet(pt.GetLabels(),"name", baseName+"_count")
		addSample(tsMap, sum, sumLbs, ty)
		addSample(tsMap, count, countLbs, ty)
		var totalCount uint64
		for le, bk := range pt.GetBuckets(){
			bucket := &prompb.Sample{
				Value:		float64(bk.Count),
				Timestamp:	time,
			}
			boundStr := strconv.FormatFloat(pt.GetExplicitBounds()[le], 'f',-1, 64)
			lbs := createLabelSet(pt.GetLabels(),"name", baseName+"_bucket", "le",boundStr)
			addSample(tsMap, bucket, lbs ,ty)
			totalCount += bk.GetCount()
		}
		infSample := &prompb.Sample{Value:float64(totalCount),Timestamp:time}
		infLbs := createLabelSet(pt.GetLabels(),"name", baseName+"_bucket", "le","+Inf")
		addSample(tsMap, infSample, infLbs, ty)
	}
	return nil
}

// handleSummaryMetric processes data points in a single OTLP summary metric by mapping the sum, count and each
// quantile of every data point as a Sample, and adding each Sample to its corresponding TimeSeries.
// tsMap and metric cannot be nil.
func (ce *cortexExporter) handleSummaryMetric(tsMap map[string]*prompb.TimeSeries, metric *otlp.Metric) error {
	if metric.SummaryDataPoints == nil {
		return fmt.Errorf("invalid metric type: wants summary points")
	}

	for _, pt := range metric.SummaryDataPoints {
		time := int64(pt.GetTimeUnixNano())
		ty := metric.GetMetricDescriptor().GetType()
		baseName := getPromMetricName(metric.GetMetricDescriptor(),ce.namespace)
		sum := &prompb.Sample{
			Value:		pt.GetSum(),
			Timestamp:	time,
		}
		count := &prompb.Sample{
			Value:		float64(pt.GetCount()),
			Timestamp:	time,
		}
		sumLbs := createLabelSet(pt.GetLabels(),"name", baseName+"_sum")
		countLbs := createLabelSet(pt.GetLabels(),"name", baseName+"_count")
		addSample(tsMap, sum, sumLbs, ty)
		addSample(tsMap, count, countLbs, ty)
		for _, qt := range pt.GetPercentileValues(){
			quantile := &prompb.Sample{
				Value:		float64(qt.Value),
				Timestamp:	time,
			}
			qtStr := strconv.FormatFloat(qt.Percentile, 'f',-1, 64)
			qtLbs := createLabelSet(pt.GetLabels(),"name", baseName, "quantile", qtStr)
			addSample(tsMap, quantile, qtLbs, ty)
		}
	}
	return nil

}

// newCortexExporter initializes a new cortexExporter instance and sets fields accordingly.
// client parameter cannot be nil.
func newCortexExporter(ns string, ep string, client *http.Client) *cortexExporter {
	return &cortexExporter{
		namespace: 	ns,
		endpoint:  	ep,
		client:    	client,
		wg:			new(sync.WaitGroup),
		closeChan:  make(chan struct{}),
	}
}

// shutdown stops the exporter from accepting incoming calls(and return error), and wait for current export operations
// to finish before returning
func (ce *cortexExporter)shutdown(context.Context) error{
	close(ce.closeChan)
	ce.wg.Wait()
	return nil
}

// pushMetrics converts metrics to Cortex TimeSeries and send to remote endpoint. It maintain a map of TimeSeries,
// validates and handles each individual metric, adding the converted TimeSeries to the map, and finally
// exports the map.
func (ce *cortexExporter) pushMetrics(ctx context.Context, md pdata.Metrics) (int, error) {
	ce.wg.Add(1)
	defer ce.wg.Done()
	select{
	case <-ce.closeChan:
		return pdatautil.MetricCount(md),fmt.Errorf("shutdown has been called")
	default:
		tsMap := map[string]*prompb.TimeSeries{}
		dropped := 0
		errStrings := []string{}
		rms := data.MetricDataToOtlp(pdatautil.MetricsToInternalMetrics(md))
		for _, r := range rms {
			// TODO add resource attributes as labels
			for _, inst := range r.InstrumentationLibraryMetrics {
				//TODO add instrumentation library information as labels
				for _, m := range inst.Metrics {
					ok := validateMetrics(m.MetricDescriptor)
					if !ok {
						dropped++
						errStrings = append(errStrings, "invalid temporality and type combination")
						continue
					}
					switch m.GetMetricDescriptor().GetType() {
					case otlp.MetricDescriptor_MONOTONIC_INT64, otlp.MetricDescriptor_INT64,
						otlp.MetricDescriptor_MONOTONIC_DOUBLE, otlp.MetricDescriptor_DOUBLE:
							ce.handleScalarMetric(tsMap,m)
					case otlp.MetricDescriptor_HISTOGRAM:
							ce.handleHistogramMetric(tsMap,m)
					case otlp.MetricDescriptor_SUMMARY:
							ce.handleSummaryMetric(tsMap,m)
					default:
						dropped++
						errStrings = append(errStrings, "invalid type")
						continue
					}
				}
			}
		}
		if(dropped != 0) {
			return dropped, fmt.Errorf(strings.Join(errStrings, "\n"))
		}

		if err := ce.export(ctx,tsMap); err != nil {
			return pdatautil.MetricCount(md), err
		}

		return 0, nil
	}
}

// export sends TimeSeries in tsMap to a Cortex Gateway
func (ce *cortexExporter) export(ctx context.Context, tsMap map[string]*prompb.TimeSeries) error {
	return nil
}

// getPromMetricName creates a Prometheus metric name by attaching namespace prefix, unit, and _total suffix
func getPromMetricName(desc *otlp.MetricDescriptor, ns string) string {
	if desc == nil {
		return ""
	}
	isCounter := desc.Type == otlp.MetricDescriptor_MONOTONIC_INT64 ||
			desc.Type == otlp.MetricDescriptor_MONOTONIC_DOUBLE
	b := strings.Builder{}
	fmt.Fprintf(&b, ns)
	if b.Len() > 0 {
		fmt.Fprintf(&b, "_")
	}
	fmt.Fprintf(&b, desc.GetName())

	if b.Len() > 0 && len(desc.GetUnit()) > 0{
		fmt.Fprintf(&b, "_")
		fmt.Fprintf(&b, desc.GetUnit())
	}

	if b.Len()>0 && isCounter {
		fmt.Fprintf(&b, "_")
		fmt.Fprintf(&b, "total")
	}
	return sanitize(b.String())
}

// copied from prometheus-go-metric-exporter
// sanitize replaces non-alphanumeric characters with underscores in s.
func sanitize(s string) string {
	if len(s) == 0 {
		return s
	}

	// Note: No length limit for label keys because Prometheus doesn't
	// define a length limit, thus we should NOT be truncating label keys.
	// See https://github.com/orijtech/prometheus-go-metrics-exporter/issues/4.

	s = strings.Map(sanitizeRune, s)
	if unicode.IsDigit(rune(s[0])) {
		s = "key_" + s
	}
	if s[0] == '_' {
		s = "key" + s
	}
	return s
}

// copied from prometheus-go-metric-exporter
// sanitizeRune converts anything that is not a letter or digit to an underscore
func sanitizeRune(r rune) rune {
	if unicode.IsLetter(r) || unicode.IsDigit(r) {
		return r
	}
	// Everything else turns into an underscore
	return '_'
}
