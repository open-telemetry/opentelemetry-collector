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

// Sample contains a simple client that periodically makes a simple http request
// to a server and exports to the OpenTelemetry service.
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
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"google.golang.org/grpc"
)

// Initializes an OTLP exporter, and configures the corresponding trace and
// metric providers.
func initProvider() func() {
	ctx := context.Background()

	otelAgentAddr, ok := os.LookupEnv("OTEL_EXPORTER_OTLP_ENDPOINT")
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
			semconv.ServiceNameKey.String("demo-client"),
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
		cxt, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		if err := traceExp.Shutdown(cxt); err != nil {
			otel.Handle(err)
		}
		// pushes any last exports to the receiver
		if err := pusher.Stop(cxt); err != nil {
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

	tracer := otel.Tracer("demo-client-tracer")
	meter := global.Meter("demo-client-meter")

	method, _ := baggage.NewMember("method", "repl")
	client, _ := baggage.NewMember("client", "cli")
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
			"demo_client/request_latency",
			metric.WithDescription("The latency of requests processed"),
		)

	// TODO: Use a view to just count number of measurements for requestLatency when available.
	requestCount := metric.Must(meter).
		NewInt64Counter(
			"demo_client/request_counts",
			metric.WithDescription("The number of requests processed"),
		)

	lineLengths := metric.Must(meter).
		NewInt64ValueRecorder(
			"demo_client/line_lengths",
			metric.WithDescription("The lengths of the various lines in"),
		)

	// TODO: Use a view to just count number of measurements for lineLengths when available.
	lineCounts := metric.Must(meter).
		NewInt64Counter(
			"demo_client/line_counts",
			metric.WithDescription("The counts of the lines in"),
		)

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
			meter.RecordBatch(
				ctx,
				commonLabels,
				lineCounts.Measurement(1),
				lineLengths.Measurement(randLineLength),
			)
			fmt.Printf("#%d: LineLength: %dBy\n", i, randLineLength)
		}

		meter.RecordBatch(
			ctx,
			commonLabels,
			requestLatency.Measurement(latencyMs),
			requestCount.Measurement(1),
		)

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
		handleErr(err, "failed to http request")
	}

	// All requests made with this client will create spans.
	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	res.Body.Close()
}
