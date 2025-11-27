// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package component_test

import (
	"fmt"

	"go.opentelemetry.io/collector/component"
)

func ExampleNewType() {
	t, err := component.NewType("exampleexporter")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(t.String())

	// Output:
	// exampleexporter
}

func ExampleMustNewType() {
	t := component.MustNewType("examplereceiver")
	fmt.Println(t.String())

	// Output:
	// examplereceiver
}

func ExampleNewID() {
	t := component.MustNewType("exampleprocessor")
	id := component.NewID(t)
	fmt.Println(id.String())

	// Output:
	// exampleprocessor
}

func ExampleMustNewID() {
	id := component.MustNewID("examplereceiver")
	fmt.Println(id.String())

	// Output:
	// examplereceiver
}

func ExampleNewIDWithName() {
	t := component.MustNewType("exampleexporter")
	id := component.NewIDWithName(t, "customname")
	fmt.Println(id.String())

	// Output:
	// exampleexporter/customname
}

func ExampleMustNewIDWithName() {
	id := component.MustNewIDWithName("exampleprocessor", "customproc")
	fmt.Println(id.String())

	// Output:
	// exampleprocessor/customproc
}
