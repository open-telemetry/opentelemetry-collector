package cortexexporter

import (
	"fmt"
	"github.com/prometheus/prometheus/prompb"
	common "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/common/v1"
	otlp "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/metrics/v1"
	"sort"
	"strings"
	"log"
	"unicode"
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
