// Copyright 2019, OpenCensus Authors
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

// Package unisvc implements Open Telemetry Service that collects stats
// and traces and exports to a configured backend.
package unisvc

import (
	"log"

	"github.com/census-instrumentation/opencensus-service/cmd/occollector/app/collector"
)

// Run the unified telemetry service.
func Run() {
	if err := collector.App.StartUnified(); err != nil {
		log.Fatalf("Failed to run the service: %v", err)
	}
}
