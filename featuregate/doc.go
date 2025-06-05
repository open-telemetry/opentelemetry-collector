// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package featuregate provides mechanisms for feature flagging in the OpenTelemetry Collector.
//
// Example (Creating a feature gate and printing its properties):
//
//	gate := featuregate.GlobalRegistry().MustRegister(
//		"example.feature",
//		featuregate.StageAlpha,
//		featuregate.WithRegisterDescription("Example feature gate for demonstration."),
//		featuregate.WithRegisterReferenceURL("https://example.com/feature"),
//		featuregate.WithRegisterFromVersion("v0.1.0"),
//	)
//	fmt.Println("ID:", gate.ID())
//	fmt.Println("Description:", gate.Description())
//	fmt.Println("Stage:", gate.Stage())
//	fmt.Println("ReferenceURL:", gate.ReferenceURL())
//	fmt.Println("FromVersion:", gate.FromVersion())
//	fmt.Println("IsEnabled:", gate.IsEnabled())
//
// Example (Registering, setting, and visiting feature gates):
//
//	reg := featuregate.NewRegistry()
//	gate := reg.MustRegister("example.feature2", featuregate.StageAlpha, featuregate.WithRegisterDescription("Another example."))
//	_ = reg.Set(gate.ID(), true) // Enable the gate
//	reg.VisitAll(func(g *featuregate.Gate) {
//		fmt.Printf("Gate: %s, Enabled: %v, Description: %s\n", g.ID(), g.IsEnabled(), g.Description())
//	})
//
package featuregate
