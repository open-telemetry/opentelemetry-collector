// Copyright 2019, OpenTelemetry Authors
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

package internal

import (
	"testing"
)

type exportNameTestCase struct {
	promName   string
	exportName string
}

func Test_exportName(t *testing.T) {
	exportPrefix := ""
	metricExportNameMap := map[string]string{}
	tests := []exportNameTestCase{
		{"prometheus_operations", "prometheus_operations"},
		{"prometheus_operations_errors", "prometheus_operations_errors"},
		{"prometheus_runtime_operations_latency_ms", "prometheus_runtime_operations_latency_ms"},
	}
	for _, testCase := range tests {
		exportName := exportName(testCase.promName, metricExportNameMap, exportPrefix)
		if testCase.exportName != exportName {
			t.Errorf("expected %v, actual %v\n", testCase.exportName, exportName)
		}
	}
}

func Test_exportNameWithPrefix(t *testing.T) {
	exportPrefix := "otel.io/prom/"
	metricExportNameMap := map[string]string{}
	tests := []exportNameTestCase{
		{"prometheus_operations", "otel.io/prom/prometheus_operations"},
		{"prometheus_operations_errors", "otel.io/prom/prometheus_operations_errors"},
		{"prometheus_runtime_operations_latency_ms", "otel.io/prom/prometheus_runtime_operations_latency_ms"},
	}
	for _, testCase := range tests {
		exportName := exportName(testCase.promName, metricExportNameMap, exportPrefix)
		if testCase.exportName != exportName {
			t.Errorf("expected %v, actual %v\n", testCase.exportName, exportName)
		}
	}
}

func Test_exportNameWithMap(t *testing.T) {
	exportPrefix := ""
	metricExportNameMap := map[string]string{
		"prometheus_operations":                    "operations",
		"prometheus_operations_errors":             "operations_errors",
		"prometheus_runtime_operations_latency_ms": "operations_runtime_latency",
	}
	tests := []exportNameTestCase{
		{"prometheus_operations", "operations"},
		{"prometheus_operations_errors", "operations_errors"},
		{"prometheus_runtime_operations_latency_ms", "operations_runtime_latency"},
	}
	for _, testCase := range tests {
		exportName := exportName(testCase.promName, metricExportNameMap, exportPrefix)
		if testCase.exportName != exportName {
			t.Errorf("expected %v, actual %v\n", testCase.exportName, exportName)
		}
	}
}

func Test_exportNameWithMapAndPrefix(t *testing.T) {
	exportPrefix := "otel.io/prom/"
	metricExportNameMap := map[string]string{
		"prometheus_operations":                    "operations",
		"prometheus_operations_errors":             "operations_errors",
		"prometheus_runtime_operations_latency_ms": "operations_runtime_latency",
	}
	tests := []exportNameTestCase{
		{"prometheus_operations", "otel.io/prom/operations"},
		{"prometheus_operations_errors", "otel.io/prom/operations_errors"},
		{"prometheus_runtime_operations_latency_ms", "otel.io/prom/operations_runtime_latency"},
	}
	for _, testCase := range tests {
		exportName := exportName(testCase.promName, metricExportNameMap, exportPrefix)
		if testCase.exportName != exportName {
			t.Errorf("expected %v, actual %v\n", testCase.exportName, exportName)
		}
	}
}
