package zipkin_test

import (
	"context"
	"log"
	"time"

	zipkin "github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/reporter"
	httpreporter "github.com/openzipkin/zipkin-go/reporter/http"
)

func doSomeWork(context.Context) {}

func ExampleNewTracer() {
	// create a reporter to be used by the tracer
	reporter := httpreporter.NewReporter("http://localhost:9411/api/v2/spans")
	defer reporter.Close()

	// set-up the local endpoint for our service
	endpoint, err := zipkin.NewEndpoint("demoService", "172.20.23.100:80")
	if err != nil {
		log.Fatalf("unable to create local endpoint: %+v\n", err)
	}

	// set-up our sampling strategy
	sampler, err := zipkin.NewBoundarySampler(0.01, time.Now().UnixNano())
	if err != nil {
		log.Fatalf("unable to create sampler: %+v\n", err)
	}

	// initialize the tracer
	tracer, err := zipkin.NewTracer(
		reporter,
		zipkin.WithLocalEndpoint(endpoint),
		zipkin.WithSampler(sampler),
	)
	if err != nil {
		log.Fatalf("unable to create tracer: %+v\n", err)
	}

	// tracer can now be used to create spans.
	span := tracer.StartSpan("some_operation")
	// ... do some work ...
	span.Finish()

	// Output:
}

func ExampleTracerOption() {
	// initialize the tracer and use the WithNoopSpan TracerOption
	tracer, _ := zipkin.NewTracer(
		reporter.NewNoopReporter(),
		zipkin.WithNoopSpan(true),
	)

	// tracer can now be used to create spans
	span := tracer.StartSpan("some_operation")
	// ... do some work ...
	span.Finish()

	// Output:
}

func ExampleNewContext() {
	var (
		tracer, _ = zipkin.NewTracer(reporter.NewNoopReporter())
		ctx       = context.Background()
	)

	// span for this function
	span := tracer.StartSpan("ExampleNewContext")
	defer span.Finish()

	// add span to Context
	ctx = zipkin.NewContext(ctx, span)

	// pass along Context which holds the span to another function
	doSomeWork(ctx)

	// Output:
}

func ExampleSpanOption() {
	tracer, _ := zipkin.NewTracer(reporter.NewNoopReporter())

	// set-up the remote endpoint for the service we're about to call
	endpoint, err := zipkin.NewEndpoint("otherService", "172.20.23.101:80")
	if err != nil {
		log.Fatalf("unable to create remote endpoint: %+v\n", err)
	}

	// start a client side RPC span and use RemoteEndpoint SpanOption
	span := tracer.StartSpan(
		"some-operation",
		zipkin.RemoteEndpoint(endpoint),
		zipkin.Kind(model.Client),
	)
	// ... call other service ...
	span.Finish()

	// Output:
}
