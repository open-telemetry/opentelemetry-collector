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

// Sample contains a program that exports to the OpenCensus service.
package main

import (
	"context"
	"log"
	"math/rand"
	"time"

	"contrib.go.opencensus.io/exporter/ocagent"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"go.opencensus.io/trace"
)

func main() {
	oce, err := ocagent.NewExporter(ocagent.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to create ocagent-exporter: %v", err)
	}
	trace.RegisterExporter(oce)
	view.RegisterExporter(oce)

	trace.ApplyConfig(trace.Config{
		DefaultSampler: trace.AlwaysSample(),
	})

	// Some stats
	keyClient, _ := tag.NewKey("client")
	keyMethod, _ := tag.NewKey("method")

	mLatencyMs := stats.Float64("latency", "The latency in milliseconds", "ms")
	views := []*view.View{
		{
			Name:        "opdemo/latency",
			Description: "The various counts",
			Measure:     mLatencyMs,
			Aggregation: view.Distribution(0, 10, 50, 100, 200, 400, 800, 1000, 1400, 2000, 5000, 10000),
			TagKeys:     []tag.Key{keyClient, keyMethod},
		},
		{
			Name:        "opdemo/process_counts",
			Description: "The various counts",
			Measure:     mLatencyMs,
			Aggregation: view.Count(),
			TagKeys:     []tag.Key{keyClient, keyMethod},
		},
	}

	if err := view.Register(views...); err != nil {
		log.Fatalf("Failed to register views for metrics: %v", err)
	}

	ctx, _ := tag.New(context.Background(), tag.Insert(keyMethod, "repl"), tag.Insert(keyClient, "cli"))
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for {
		startTime := time.Now()
		_, span := trace.StartSpan(context.Background(), "Foo")
		time.Sleep(time.Duration(rng.Int63n(13000)) * time.Millisecond)
		span.End()
		latencyMs := float64(time.Since(startTime)) / 1e6
		stats.Record(ctx, mLatencyMs.M(latencyMs))
	}
}
