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

// Program otelcol is the OpenTelemetry Collector that collects stats
// and traces and exports to a configured backend.
package main

import (
	"log"

	"go.opentelemetry.io/collector/internal/version"
	"go.opentelemetry.io/collector/service"
	"go.opentelemetry.io/collector/service/defaultcomponents"
)

func main() {
	handleErr := func(message string, err error) {
		if err != nil {
			log.Fatalf("%s: %v", message, err)
		}
	}

	factories, err := defaultcomponents.Components()
	handleErr("Failed to build default components", err)

	info := service.ApplicationStartInfo{
		ExeName:  "otelcol",
		LongName: "OpenTelemetry Collector",
		Version:  version.Version,
		GitHash:  version.GitHash,
	}

	svc, err := service.New(service.Parameters{ApplicationStartInfo: info, Factories: factories})
	handleErr("Failed to construct the application", err)

	err = svc.Start()
	handleErr("Application run finished with error", err)
}
