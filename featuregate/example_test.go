// Copyright The OpenTelemetry Authors
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
package featuregate_test

import (
	"fmt"

	"go.opentelemetry.io/collector/featuregate"
)

// Example_createFeatureGate demonstrates creating a feature gate and printing its properties.
func Example_createFeatureGate() {
	gate := featuregate.GlobalRegistry().MustRegister(
		"example.feature",
		featuregate.StageAlpha,
		featuregate.WithRegisterDescription("Example feature gate for demonstration."),
		featuregate.WithRegisterReferenceURL("https://example.com/feature"),
		featuregate.WithRegisterFromVersion("v0.1.0"),
	)
	fmt.Println("ID:", gate.ID())
	fmt.Println("Description:", gate.Description())
	fmt.Println("Stage:", gate.Stage())
	fmt.Println("ReferenceURL:", gate.ReferenceURL())
	fmt.Println("FromVersion:", gate.FromVersion())
	fmt.Println("IsEnabled:", gate.IsEnabled())
	// Output:
	// ID: example.feature
	// Description: Example feature gate for demonstration.
	// Stage: Alpha
	// ReferenceURL: https://example.com/feature
	// FromVersion: v0.1.0
	// IsEnabled: false
}

// Example_registrySetAndVisit demonstrates registering, setting, and visiting feature gates.
func Example_registrySetAndVisit() {
	reg := featuregate.NewRegistry()
	gate := reg.MustRegister("example.feature2", featuregate.StageAlpha, featuregate.WithRegisterDescription("Another example."))
	_ = reg.Set(gate.ID(), true) // Enable the gate
	reg.VisitAll(func(g *featuregate.Gate) {
		fmt.Printf("Gate: %s, Enabled: %v, Description: %s\n", g.ID(), g.IsEnabled(), g.Description())
	})
	// Output:
	// Gate: example.feature2, Enabled: true, Description: Another example.
}
