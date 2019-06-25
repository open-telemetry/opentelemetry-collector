package testbed

import (
	"encoding/binary"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"contrib.go.opencensus.io/exporter/jaeger"
	"go.opencensus.io/trace"
)

// LoadGenerator is a simple load generator.
type LoadGenerator struct {
	exporter *jaeger.Exporter

	TracesSent uint64
	SpansSent  uint64

	stopOnce   sync.Once
	stopWait   sync.WaitGroup
	stopSignal chan struct{}

	options LoadOptions
}

// LoadOptions defines the options to use for generating the load.
type LoadOptions struct {
	// SpansPerSecond specifies how many spans to generate each second.
	SpansPerSecond uint

	// SpansPerTrace specifies how many spans per trace to generate. Should be equal or greater than zero.
	// The number of traces generated per second will be SpansPerSecond/SpansPerTrace.
	SpansPerTrace uint

	// Attributes to add to each generated span. Can be empty.
	Attributes map[string]interface{}
}

// NewLoadGenerator creates a load generator.
func NewLoadGenerator() (*LoadGenerator, error) {
	lg := &LoadGenerator{}

	lg.stopSignal = make(chan struct{})

	opts := jaeger.Options{
		CollectorEndpoint: "http://localhost:14268/api/traces",
		Process: jaeger.Process{
			ServiceName: "load-generator",
		},
	}

	var err error
	lg.exporter, err = jaeger.NewExporter(opts)
	if err != nil {
		return nil, err
	}

	return lg, nil
}

// Start the load.
func (lg *LoadGenerator) Start(options LoadOptions) {
	lg.options = options

	if lg.options.SpansPerTrace == 0 {
		// 10 spans per trace by default.
		lg.options.SpansPerTrace = 10
	}

	log.Printf("Starting load generator at %d spans/sec.", lg.options.SpansPerSecond)

	// Indicate that generation is in progress.
	lg.stopWait.Add(1)

	// Begin generation
	go lg.generate()
}

// Stop the load.
func (lg *LoadGenerator) Stop() {
	lg.stopOnce.Do(func() {
		// Signal generate() to stop.
		close(lg.stopSignal)

		// Wait for it to stop.
		lg.stopWait.Wait()

		// Print stats.
		log.Printf("Stopped generator. %s", lg.GetStats())
	})
}

// GetStats returns the stats as a printable string.
func (lg *LoadGenerator) GetStats() string {
	return fmt.Sprintf("Sent:%5d spans", atomic.LoadUint64(&lg.SpansSent))
}

func (lg *LoadGenerator) generate() {
	// Indicate that generation is done at the end
	defer lg.stopWait.Done()

	if lg.options.SpansPerSecond <= 0 {
		return
	}

	t := time.NewTicker(time.Second / time.Duration(lg.options.SpansPerSecond/lg.options.SpansPerTrace))
	defer t.Stop()
	done := false
	for !done {
		select {
		case <-t.C:
			lg.generateTrace()

		case <-lg.stopSignal:
			done = true
		}
	}
	// Send all pending generated spans
	lg.exporter.Flush()
}

func (lg *LoadGenerator) generateTrace() {

	traceID := atomic.AddUint64(&lg.TracesSent, 1)
	for i := uint(0); i < lg.options.SpansPerTrace; i++ {

		startTime := time.Now()

		spanID := atomic.AddUint64(&lg.SpansSent, 1)

		// Create a span.
		span := &trace.SpanData{
			SpanContext: trace.SpanContext{
				TraceID: generateTraceID(traceID),
				SpanID:  generateSpanID(spanID),
			},
			Name:     "load-generator-span",
			SpanKind: trace.SpanKindClient,
			Attributes: map[string]interface{}{
				"load_generator.span_seq_num":  int64(spanID),
				"load_generator.trace_seq_num": int64(traceID),
			},
			StartTime: startTime,
			EndTime:   startTime.Add(time.Duration(time.Millisecond)),
		}

		// Append attributes.
		for k, v := range lg.options.Attributes {
			span.Attributes[k] = v
		}

		lg.exporter.ExportSpan(span)
	}
}

func generateTraceID(id uint64) trace.TraceID {
	var traceID trace.TraceID
	binary.PutUvarint(traceID[:], id)
	return traceID
}

func generateSpanID(id uint64) trace.SpanID {
	var spanID trace.SpanID
	binary.PutUvarint(spanID[:], id)
	return spanID
}
