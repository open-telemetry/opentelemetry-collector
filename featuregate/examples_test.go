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
		"featuregate.example.gate",
		featuregate.StageAlpha,
		featuregate.WithRegisterDescription("Example gate"),
	)

	// By default an alpha feature gate is disabled.
	fmt.Println(gate.IsEnabled())
	if err := reg.Set(gate.ID(), true); err != nil {
		fmt.Println("error:", err)
		return
	}
	// Alpha feature gates can be modified so the gate is now enabled.
	fmt.Println(gate.IsEnabled())

	// Output:
	// false
	// true
}

func ExampleRegistry_RegisterFlags() {
	reg := featuregate.NewRegistry()
	alphaGate := reg.MustRegister("featuregate.example.alpha", featuregate.StageAlpha)
	betaGate := reg.MustRegister("featuregate.example.beta", featuregate.StageBeta)

	fs := flag.NewFlagSet("example", flag.ContinueOnError)
	// RegisterFlags registers the `--feature-gates` flag on the FlagSet.
	reg.RegisterFlags(fs)
	if err := fs.Parse([]string{"--feature-gates=featuregate.example.alpha,-featuregate.example.beta"}); err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Printf("featuregate.example.alpha=%v\n", alphaGate.IsEnabled())
	fmt.Printf("featuregate.example.beta=%v\n", betaGate.IsEnabled())

	// Output:
	// featuregate.example.alpha=true
	// featuregate.example.beta=false
}
