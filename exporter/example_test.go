// Copyright 2018, OpenCensus Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package exporter_test

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"contrib.go.opencensus.io/exporter/ocagent"
	"contrib.go.opencensus.io/exporter/stackdriver"
	"go.opencensus.io/exporter/jaeger"
	"go.opencensus.io/trace"

	"github.com/census-instrumentation/opencensus-service/exporter"
	"github.com/census-instrumentation/opencensus-service/receiver/opencensus"
)

func Example_endToEndExporting() {
	// The server runs on the agent e.g. cmd/ocagent
	srvPort, srvCleanup := runServer()
	defer srvCleanup()

	// The client applications/programs will be running on their own.
	if err := runClientApplications(srvPort); err != nil {
		log.Fatalf("Failed to run client application: %v", err)
	}
}

func runClientApplications(port uint16) error {
	address := fmt.Sprintf("localhost:%d", port)
	oce, err := ocagent.NewExporter(ocagent.WithAddress(address), ocagent.WithInsecure())
	if err != nil {
		return err
	}
	trace.RegisterExporter(oce)

	// For this demo we'll generate traffic and always sample
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})

	var wg sync.WaitGroup
	// Generate some traffic
	for i := 0; i < 20; i++ {
		// This shows what running client applications would be,
		// in various languages producing traces simultaneously.
		wg.Add(1)
		rms := time.Duration(2+rand.Intn(180)) * time.Millisecond
		go func(id int64, loadTime time.Duration) {
			ctx, span := trace.StartSpan(context.Background(), fmt.Sprintf("ParentSpan-%d", id))
			defer span.End()

			if id%2 == 0 {
				_, cSpan := trace.StartSpan(ctx, fmt.Sprintf("ChildSpan-%d", id))
				cSpan.Annotate(nil, "This is a child")
				cSpan.End()
			} else {
				span.Annotatef([]trace.Attribute{
					trace.Int64Attribute("i", id),
				}, "ParentSpan annotation")
			}
			<-time.After(loadTime)
		}(int64(i), rms)
	}

	// Wait for all the "client programs" to complete.
	wg.Wait()
	oce.Flush()

	<-time.After(4 * time.Second)

	return nil
}

func runServer() (port uint16, closeFn func() error) {
	// The first phase will feature how cmd/ocagent will
	// parse the requested trace OpenCensus exporters.
	sde, err := stackdriver.NewExporter(stackdriver.Options{ProjectID: "census-demos"})
	if err != nil {
		log.Fatalf("Failed to start Stackdriver Trace exporter: %v", err)
	}

	je, err := jaeger.NewExporter(jaeger.Options{
		Endpoint:    "http://localhost:14268/api/traces",
		ServiceName: "plumbing-demo",
	})
	if err != nil {
		log.Fatalf("Failed to start Jaeger Trace exporter: %v", err)
	}

	port = 55679
	addr := fmt.Sprintf("localhost:%d", port)

	// Now run the OpenCensus receiver which will receive traces from the client applications
	// in the various languages instrumented with OpenCensus.
	ocr, err := opencensus.New(addr)
	if err != nil {
		log.Fatalf("Failed to create the OpenCensus receiver: %v", err)
	}

	// After each of the exporters have been created, create the common spansink.Sink.
	commonSpanSink := exporter.OCExportersToTraceExporter(sde, je)

	if err := ocr.StartTraceReception(context.Background(), commonSpanSink); err != nil {
		log.Fatalf("Failed to run the OpenCensus TraceReceiver: %v", err)
	}
	closeFn = ocr.Stop
	return port, closeFn
}
