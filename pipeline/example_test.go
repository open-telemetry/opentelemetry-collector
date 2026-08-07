// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pipeline_test

import (
	"fmt"

	"go.opentelemetry.io/collector/pipeline"
)

func ExampleNewIDWithName() {
	id := pipeline.NewIDWithName(pipeline.SignalTraces, "gateway")

	fmt.Println(id)
	fmt.Println(id.Signal())
	fmt.Println(id.Name())

	// Output:
	// traces/gateway
	// traces
	// gateway
}

func ExampleID_UnmarshalText() {
	var id pipeline.ID
	if err := id.UnmarshalText([]byte("metrics/host")); err != nil {
		panic(err)
	}

	fmt.Printf("signal: %s\n", id.Signal())
	fmt.Printf("name: %s\n", id.Name())

	// Output:
	// signal: metrics
	// name: host
}
