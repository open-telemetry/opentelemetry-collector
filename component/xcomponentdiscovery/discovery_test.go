// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xcomponentdiscovery

import "testing"

func TestRuleIsEmpty(t *testing.T) {
	tests := []struct {
		name string
		rule Rule
		want bool
	}{
		{name: "zero value", rule: Rule{}, want: true},
		{name: "config only", rule: Rule{DefaultConfig: map[string]any{"a": 1}}, want: true},
		{name: "empty endpoints", rule: Rule{Endpoints: map[EndpointType]EndpointRule{}}, want: true},
		{name: "one endpoint type", rule: Rule{Endpoints: map[EndpointType]EndpointRule{"port": {Rule: "port == 6379"}}}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rule.IsEmpty(); got != tt.want {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}
