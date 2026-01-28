// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package featuregate_test

import (
	"flag"
	"fmt"

	"go.opentelemetry.io/collector/featuregate"
)

func ExampleRegistry_Register() {
	reg := featuregate.NewRegistry()
	gate := reg.MustRegister(
		"example.gate",
		featuregate.StageAlpha,
		featuregate.WithRegisterDescription("Example gate"),
	)

	fmt.Println(gate.IsEnabled())
	if err := reg.Set(gate.ID(), true); err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(gate.IsEnabled())

	// Output:
	// false
	// true
}

func ExampleRegistry_RegisterFlags() {
	reg := featuregate.NewRegistry()
	alphaGate := reg.MustRegister("alpha", featuregate.StageAlpha)
	betaGate := reg.MustRegister("beta", featuregate.StageBeta)

	fs := flag.NewFlagSet("example", flag.ContinueOnError)
	reg.RegisterFlags(fs)
	if err := fs.Parse([]string{"--feature-gates=alpha,-beta"}); err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Printf("featuregate.example.alpha=%v\n", alphaGate.IsEnabled())
	fmt.Printf("featuregate.example.beta=%v\n", betaGate.IsEnabled())

	// Output:
	// featuregate.example.alpha=true
	// featuregate.example.beta=false
}
