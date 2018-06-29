package ddtrace_test

import (
	"io/ioutil"
	"log"

	opentracing "github.com/opentracing/opentracing-go"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/mocktracer"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/opentracer"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

// The below example illustrates a simple use case using the "tracer" package,
// our native Datadog APM tracing client integration. For thorough documentation
// and further examples, visit its own godoc page.
func Example_datadog() {
	// Start the tracer and defer the Stop method.
	tracer.Start(tracer.WithAgentAddr("host:port"))
	defer tracer.Stop()

	// Start a root span.
	span := tracer.StartSpan("get.data")
	defer span.Finish()

	// Create a child of it, computing the time needed to read a file.
	child := tracer.StartSpan("read.file", tracer.ChildOf(span.Context()))
	child.SetTag(ext.ResourceName, "test.json")

	// Perform an operation.
	_, err := ioutil.ReadFile("~/test.json")

	// We may finish the child span using the returned error. If it's
	// nil, it will be disregarded.
	child.Finish(tracer.WithError(err))
	if err != nil {
		log.Fatal(err)
	}
}

// The below example illustrates how to set up an opentracing.Tracer using Datadog's
// tracer.
func Example_opentracing() {
	// Start a Datadog tracer, optionally providing a set of options,
	// returning an opentracing.Tracer which wraps it.
	t := opentracer.New(tracer.WithAgentAddr("host:port"))

	// Use it with the Opentracing API. The (already started) Datadog tracer
	// may be used in parallel with the Opentracing API if desired.
	opentracing.SetGlobalTracer(t)
}

// The code below illustrates a scenario of how one could use a mock tracer in tests
// to assert that spans are created correctly.
func Example_mocking() {
	// Start the mock tracer.
	mt := mocktracer.Start()
	defer mt.Stop()

	// Create a test span (this usually happens in your code).
	span := tracer.StartSpan("test.span")
	span.Finish()

	// Query the mock tracer for finished spans.
	spans := mt.FinishedSpans()
	if len(spans) != 1 {
		// should only have 1 span
	}

	// Run assertions...
}
