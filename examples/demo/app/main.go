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

// Sample contains a program that exports to the OpenTelemetry service.
package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/propagation"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv/v1.4.0"
	"google.golang.org/grpc"
)

// Initializes an OTLP exporter, and configures the corresponding trace and
// metric providers.
func initProvider() func() {
	ctx := context.Background()

	otelAgentAddr, ok := os.LookupEnv("OTEL_AGENT_ENDPOINT")
	if !ok {
		otelAgentAddr = "0.0.0.0:4317"
	}

	metricClient := otlpmetricgrpc.NewClient(
		otlpmetricgrpc.WithInsecure(),
		otlpmetricgrpc.WithEndpoint(otelAgentAddr))
	metricExp, err := otlpmetric.New(ctx, metricClient)
	handleErr(err, "Failed to create the collector metric exporter")

	pusher := controller.New(
		processor.New(
			simple.NewWithExactDistribution(),
			metricExp,
		),
		controller.WithExporter(metricExp),
		controller.WithCollectPeriod(2*time.Second),
	)
	global.SetMeterProvider(pusher.MeterProvider())

	err = pusher.Start(ctx)
	handleErr(err, "Failed to start metric pusher")

	traceClient := otlptracegrpc.NewClient(
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(otelAgentAddr),
		otlptracegrpc.WithDialOption(grpc.WithBlock()))
	traceExp, err := otlptrace.New(ctx, traceClient)
	handleErr(err, "Failed to create the collector trace exporter")

	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithAttributes(
			// the service name used to display traces in backends
			semconv.ServiceNameKey.String("test-service"),
		),
	)
	handleErr(err, "failed to create resource")

	bsp := sdktrace.NewBatchSpanProcessor(traceExp)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	// set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.TraceContext{})
	otel.SetTracerProvider(tracerProvider)

	return func() {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		if err := traceExp.Shutdown(ctx); err != nil {
			otel.Handle(err)
		}
		// pushes any last exports to the receiver
		if err := pusher.Stop(ctx); err != nil {
			otel.Handle(err)
		}
	}
}

func handleErr(err error, message string) {
	if err != nil {
		log.Fatalf("%s: %v", message, err)
	}
}

func main() {
	shutdown := initProvider()
	defer shutdown()

	tracer := otel.Tracer("test-tracer")
	meter := global.Meter("test-meter")

	method, _ := baggage.NewMember("method","repl")
	client, _ := baggage.NewMember("client","cli")
	bag, _ := baggage.New(method, client)

	// labels represent additional key-value descriptors that can be bound to a
	// metric observer or recorder.
	// TODO: Use baggage when supported to extract labels from baggage.
	commonLabels := []attribute.KeyValue{
		attribute.String("method", "repl"),
		attribute.String("client", "cli"),
	}

	// Recorder metric example
	requestLatency := metric.Must(meter).
		NewFloat64ValueRecorder(
			"appdemo/request_latency",
			metric.WithDescription("The latency of requests processed"),
		).Bind(commonLabels...)
	defer requestLatency.Unbind()

	// TODO: Use a view to just count number of measurements for requestLatency when available.
	requestCount := metric.Must(meter).
		NewInt64Counter(
			"appdemo/request_counts",
			metric.WithDescription("The number of requests processed"),
		).Bind(commonLabels...)
	defer requestCount.Unbind()

	lineLengths := metric.Must(meter).
		NewInt64ValueRecorder(
			"appdemo/line_lengths",
			metric.WithDescription("The lengths of the various lines in"),
		).Bind(commonLabels...)
	defer lineLengths.Unbind()

	// TODO: Use a view to just count number of measurements for lineLengths when available.
	lineCounts := metric.Must(meter).
		NewInt64Counter(
			"appdemo/line_counts",
			metric.WithDescription("The counts of the lines in"),
		).Bind(commonLabels...)
	defer lineCounts.Unbind()


	defaultCtx := baggage.ContextWithBaggage(context.Background(), bag)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for {
		startTime := time.Now()
		ctx, span := tracer.Start(defaultCtx, "ExecuteRequest")
		makeRequest(ctx)
		span.End()
		latencyMs := float64(time.Since(startTime)) / 1e6
		nr := int(rng.Int31n(7))
		for i := 0; i < nr; i++ {
			randLineLength := rng.Int63n(999)
			lineLengths.Record(ctx, randLineLength)
			lineCounts.Add(ctx, 1)
			fmt.Printf("#%d: LineLength: %dBy\n", i, randLineLength)
		}

		requestLatency.Record(ctx, latencyMs)
		requestCount.Add(ctx, 1)
		fmt.Printf("Latency: %.3fms\n", latencyMs)
		time.Sleep(time.Duration(1) * time.Second)
	}
}

func makeRequest(ctx context.Context) {

	demoServerAddr, ok := os.LookupEnv("DEMO_SERVER_ENDPOINT")
	if !ok {
		demoServerAddr = "http://0.0.0.0:7080/hello"
	}

	// Trace an HTTP client by wrapping the transport
	client := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	// Make sure we pass the context to the request to avoid broken traces.
	req, err := http.NewRequestWithContext(ctx, "GET", demoServerAddr, nil)
	if err != nil {
		fmt.Errorf("error creating http request %w", err)
	}

	// All requests made with this client will create spans.
	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	res.Body.Close()
}
