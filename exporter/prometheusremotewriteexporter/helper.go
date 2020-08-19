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

package prometheusremotewriteexporter

import (
	"log"
	"sort"
	"strings"
	"unicode"

	"github.com/prometheus/prometheus/prompb"

	common "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/common/v1"
	otlp "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/metrics/v1"
)

const (
	totalStr  = "total"
	delimeter = "_"
	keyStr    = "key"
)

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

// addSample finds a TimeSeries in tsMap that corresponds to the label set labels, and add sample to the TimeSeries; it
// creates a new TimeSeries in the map if not found. tsMap is unmodified if either of its parameters is nil.
func addSample(tsMap map[string]*prompb.TimeSeries, sample *prompb.Sample, labels []prompb.Label,
	ty otlp.MetricDescriptor_Type) {

	if sample == nil || labels == nil || tsMap == nil {
		return
	}

	sig := timeSeriesSignature(ty, &labels)
	ts, ok := tsMap[sig]

	if ok {
		ts.Samples = append(ts.Samples, *sample)
	} else {
		newTs := &prompb.TimeSeries{
			Labels:  labels,
			Samples: []prompb.Sample{*sample},
		}
		tsMap[sig] = newTs
	}
}

// timeSeries return a string signature in the form of:
// 		TYPE-label1-value1- ...  -labelN-valueN
// the label slice should not contain duplicate label names; this method sorts the slice by label name before creating
// the signature.
func timeSeriesSignature(t otlp.MetricDescriptor_Type, labels *[]prompb.Label) string {
	b := strings.Builder{}
	b.WriteString(t.String())

	sort.Sort(ByLabelName(*labels))

	for _, lb := range *labels {
		b.WriteString("-")
		b.WriteString(lb.GetName())
		b.WriteString("-")
		b.WriteString(lb.GetValue())
	}

	return b.String()
}

// createLabelSet creates a slice of Cortex Label with OTLP labels and paris of string values.
// Unpaired string value is ignored. String pairs overwrites OTLP labels if collision happens, and the overwrite is
// logged. Resultant label names are sanitized.
func createLabelSet(labels []*common.StringKeyValue, extras ...string) []prompb.Label {

	// map ensures no duplicate label name
	l := map[string]prompb.Label{}

	for _, lb := range labels {
		l[lb.Key] = prompb.Label{
			Name:  sanitize(lb.Key),
			Value: lb.Value,
		}
	}

	for i := 0; i < len(extras); i += 2 {
		if i+1 >= len(extras) {
			break
		}
		_, found := l[extras[i]]
		if found {
			log.Println("label " + extras[i] + " is overwritten. Check if Prometheus reserved labels are used.")
		}
		// internal labels should be maintained
		name := extras[i]
		if !(len(name) > 4 && name[:2] == "__" && name[len(name)-2:] == "__") {
			name = sanitize(name)
		}
		l[extras[i]] = prompb.Label{
			Name:  name,
			Value: extras[i+1],
		}
	}

	s := make([]prompb.Label, 0, len(l))

	for _, lb := range l {
		s = append(s, lb)
	}

	return s
}

// getPromMetricName creates a Prometheus metric name by attaching namespace prefix, and _total suffix for Monotonic
// metrics.
func getPromMetricName(desc *otlp.MetricDescriptor, ns string) string {

	if desc == nil {
		return ""
	}
	// whether _total suffix should be applied
	isCounter := desc.Type == otlp.MetricDescriptor_MONOTONIC_INT64 ||
		desc.Type == otlp.MetricDescriptor_MONOTONIC_DOUBLE

	b := strings.Builder{}

	b.WriteString(ns)

	if b.Len() > 0 {
		b.WriteString(delimeter)
	}
	b.WriteString(desc.GetName())

	// Including units makes two metrics with the same name and label set belong to two different TimeSeries if the
	// units are different.
	/*
		if b.Len() > 0 && len(desc.GetUnit()) > 0{
			fmt.Fprintf(&b, delimeter)
			fmt.Fprintf(&b, desc.GetUnit())
		}
	*/

	if b.Len() > 0 && isCounter {
		b.WriteString(delimeter)
		b.WriteString(totalStr)
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
		s = keyStr + delimeter + s
	}
	if s[0] == '_' {
		s = keyStr + s
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
