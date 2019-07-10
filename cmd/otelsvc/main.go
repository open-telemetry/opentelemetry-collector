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

// Program otelsvc is the Open Telemetry Service that collects stats
// and traces and exports to a configured backend.
package main

import (
	"log"

	"github.com/open-telemetry/opentelemetry-service/application"
	_ "github.com/open-telemetry/opentelemetry-service/receiver/vmmetricsreceiver"
)

func main() {
	if err := application.App.StartUnified(); err != nil {
		log.Fatalf("Failed to run the service: %v", err)
	}
}
