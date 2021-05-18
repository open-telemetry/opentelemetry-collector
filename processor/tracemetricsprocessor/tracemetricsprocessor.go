// Copyright  The OpenTelemetry Authors
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

package tracemetricsprocessor

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
)

type metricKey string

type traceMetricsProcessor struct {
	logger *zap.Logger
	config Config
	ctx    context.Context

	// The starting time of the data points.
	startTime time.Time

	lock            sync.RWMutex
	metricsExporter component.MetricsExporter
	nextConsumer    consumer.Traces
	ticker          *time.Ticker

	// Active trace sessions
	traceSessions map[pdata.TraceID]*traceSession
	// A list which allows easy searching of inactive trace sessions
	traceIDList traceIDList
}

type traceSession struct {
	rootSpan  pdata.SpanID
	spans     map[pdata.SpanID]*pdata.Span
	listEntry *node
}

type traceIDList struct {
	latest *node
	oldest *node
}

type node struct {
	next               *node
	prev               *node
	traceID            pdata.TraceID
	lastSpanInsertTime time.Time
}

func newProcessor(ctx context.Context, logger *zap.Logger, config config.Exporter, nextConsumer consumer.Traces) *traceMetricsProcessor {
	logger.Info("Building traceMetricsProcessor")
	pConfig := config.(*Config)

	return &traceMetricsProcessor{
		ctx:           ctx,
		logger:        logger,
		config:        *pConfig,
		startTime:     time.Now(),
		nextConsumer:  nextConsumer,
		traceSessions: make(map[pdata.TraceID]*traceSession),
	}
}

// GetCapabilities implements the component.Processor interface.
func (tp *traceMetricsProcessor) GetCapabilities() component.ProcessorCapabilities {
	tp.logger.Info("GetCapabilities for traceMetricsProcessor")
	return component.ProcessorCapabilities{MutatesConsumedData: false}
}

// Shutdown implements the component.Component interface.
func (tp *traceMetricsProcessor) Shutdown(_ context.Context) error {
	tp.logger.Info("Shutting down traceMetricsProcessor")
	return nil
}

func (tp *traceMetricsProcessor) Start(_ context.Context, host component.Host) error {
	tp.logger.Info("Starting traceMetricsProcessor")
	exporters := host.GetExporters()

	var availableMetricsExporters []string

	// The available list of exporters come from any configured metrics pipelines' exporters.
	for k, exp := range exporters[config.MetricsDataType] {
		metricsExp, ok := exp.(component.MetricsExporter)
		if !ok {
			return fmt.Errorf("the exporter %q isn't a metrics exporter", k.Name())
		}

		availableMetricsExporters = append(availableMetricsExporters, k.Name())

		tp.logger.Debug("Looking for traceMetricsProcessor exporter from available exporters",
			zap.String("traceMetricsProcessor-exporter", tp.config.MetricsExporter),
			zap.Any("available-exporters", availableMetricsExporters),
		)
		if k.Name() == tp.config.MetricsExporter {
			tp.metricsExporter = metricsExp
			tp.logger.Info("Found exporter", zap.String("traceMetricsProcessor-exporter", tp.config.MetricsExporter))
			break
		}
	}
	go startTicker(tp)
	tp.logger.Info("Started traceMetricsProcessor")
	return nil
}

func (tp *traceMetricsProcessor) ConsumeTraces(ctx context.Context, traces pdata.Traces) error {
	tp.processTraces(traces)
	// Forward trace data unmodified.
	if err := tp.nextConsumer.ConsumeTraces(ctx, traces); err != nil {
		return err
	}
	return nil
}

func (tp *traceMetricsProcessor) processTraces(traces pdata.Traces) {
	for i := 0; i < traces.ResourceSpans().Len(); i++ {
		rs := traces.ResourceSpans().At(i)
		tp.processResourceSpan(rs)
	}
}

func (tp *traceMetricsProcessor) processResourceSpan(rs pdata.ResourceSpans) {
	ilsSlice := rs.InstrumentationLibrarySpans()
	for j := 0; j < ilsSlice.Len(); j++ {
		ils := ilsSlice.At(j)
		tp.processILSpans(ils)
	}
}

func (tp *traceMetricsProcessor) processILSpans(ils pdata.InstrumentationLibrarySpans) {
	spans := ils.Spans()
	for k := 0; k < spans.Len(); k++ {
		span := spans.At(k)
		tp.lock.Lock()
		tp.processSpan(span)
		tp.lock.Unlock()
	}
}

func (tp *traceMetricsProcessor) processSpan(span pdata.Span) {
	session := tp.getSession(span.TraceID())
	session.addSpan(&span)
	tp.traceIDList.markAsLatest(session.listEntry)
}

func (tp *traceMetricsProcessor) getSession(traceID pdata.TraceID) *traceSession {
	if v, ok := tp.traceSessions[traceID]; ok {
		return v
	}
	t := &traceSession{
		spans: make(map[pdata.SpanID]*pdata.Span),
	}
	tp.traceSessions[traceID] = t
	return t
}

func (ts *traceSession) addSpan(span *pdata.Span) {
	if span.ParentSpanID().IsEmpty() {
		ts.rootSpan = span.SpanID()
	}
	if ts.listEntry == nil {
		ts.listEntry = &node{
			traceID:            span.TraceID(),
			lastSpanInsertTime: time.Now(),
		}
	}
	ts.spans[span.SpanID()] = span
}

func (l *traceIDList) markAsLatest(n *node) {
	n.lastSpanInsertTime = time.Now()
	if l.oldest == nil && l.latest == nil {
		l.latest = n
		l.oldest = n
		return
	}
	if n == l.latest {
		return
	}
	if n.prev != nil {
		n.prev.next = n.next
		if n.next != nil {
			n.next.prev = n.prev
		} else {
			l.oldest = n.prev
		}
		n.next = nil
		n.prev = nil
	}
	n.next = l.latest
	n.next.prev = n
	l.latest = n
}

func startTicker(tp *traceMetricsProcessor) {
	tp.ticker = time.NewTicker(30 * time.Second)
	for range tp.ticker.C {
		tp.lock.Lock()
		processCompletedTraces(tp)
		tp.lock.Unlock()
	}
}

func processCompletedTraces(tp *traceMetricsProcessor) {
	toRemove := tp.traceIDList.collectNodesToRemove(time.Now().Add(time.Duration(-30) * time.Second))
	tp.logger.Info("Collected traces remove " + strconv.Itoa(len(toRemove)))
	callSum := make(map[metricKey]int64)
	for _, traceID := range toRemove {
		session := tp.traceSessions[traceID]
		btName := computeBTName(session)
		for spanID := range session.spans {
			span := session.spans[spanID]
			parentSpan := computeParentSpanName(session, span)
			tp.aggregateMetricsForSpan(btName, span, parentSpan, callSum)
		}
		delete(tp.traceSessions, traceID)
	}
	m := tp.buildMetrics(callSum)
	if err := tp.metricsExporter.ConsumeMetrics(tp.ctx, *m); err != nil {
		tp.logger.Error("")
	}

}

func computeParentSpanName(session *traceSession, span *pdata.Span) string {
	parentSpan := "nil"
	if v, ok := session.spans[span.ParentSpanID()]; ok {
		parentSpan = v.Name()
	}
	return parentSpan
}

func computeBTName(session *traceSession) string {
	btName := "unknown"
	if v, ok := session.spans[session.rootSpan]; ok {
		btName = v.Name()
	} else {
		fmt.Println("missing bt name ")
	}
	return btName
}

func (tp *traceMetricsProcessor) buildMetrics(callSum map[metricKey]int64) *pdata.Metrics {
	tp.logger.Info("publishing metrics " + strconv.Itoa(len(callSum)))
	m := pdata.NewMetrics()
	ilm := pdata.NewInstrumentationLibraryMetrics()
	rm := pdata.NewResourceMetrics()
	rm.Resource().Attributes().Insert("traceMetricsProcessorKey", pdata.NewAttributeValueString("traceMetricsProcessorVal"))
	m.ResourceMetrics().Append(rm)
	rm.InstrumentationLibraryMetrics().Append(ilm)
	ilm.InstrumentationLibrary().SetName("traceMetricsProcessor")
	tp.collectCallMetrics(ilm, callSum)
	return &m
}

// collectCallMetrics collects the raw call count metrics, writing the data
// into the given instrumentation library metrics.
func (tp *traceMetricsProcessor) collectCallMetrics(ilm pdata.InstrumentationLibraryMetrics, callSum map[metricKey]int64) {
	for key := range callSum {
		mCalls := pdata.NewMetric()
		ilm.Metrics().Append(mCalls)
		mCalls.SetDataType(pdata.MetricDataTypeIntSum)
		mCalls.SetName("bt_cpm")
		mCalls.IntSum().SetIsMonotonic(true)
		mCalls.IntSum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)

		dpCalls := pdata.NewIntDataPoint()
		mCalls.IntSum().DataPoints().Append(dpCalls)
		dpCalls.SetStartTime(pdata.TimestampFromTime(time.Now().Add(time.Duration(-30) * time.Second)))
		dpCalls.SetTimestamp(pdata.TimestampFromTime(time.Now()))
		dpCalls.SetValue(callSum[key])
		dpCalls.LabelsMap().InitFromMap(stringToMap(string(key)))
	}
}

func (tp *traceMetricsProcessor) aggregateMetricsForSpan(btName string, span *pdata.Span, parentSpan string, callSum map[metricKey]int64) {
	d := map[string]string{
		"bt":         btName,
		"span":       span.Name(),
		"parentSpan": parentSpan,
	}
	metricName := mapToString(d)
	callSum[metricKey(metricName)]++
}

func (l *traceIDList) collectNodesToRemove(cutoffTime time.Time) []pdata.TraceID {
	var ids []pdata.TraceID
	current := l.oldest
	for current != nil && current.lastSpanInsertTime.Before(cutoffTime) {
		l.oldest = current.prev
		if current.prev != nil {
			current.prev.next = nil
		}
		ids = append(ids, current.traceID)
		current = current.prev
	}
	return ids
}

func mapToString(m map[string]string) string {
	b := new(bytes.Buffer)
	for key, value := range m {
		fmt.Fprintf(b, "%s=%s,", key, value)
	}
	return strings.ToLower(b.String())
}

func stringToMap(s string) map[string]string {
	entries := strings.Split(s, ",")
	m := make(map[string]string)
	for _, e := range entries {
		parts := strings.Split(e, "=")
		if len(parts) > 1 {
			m[parts[0]] = parts[1]
		}
	}
	return m
}
