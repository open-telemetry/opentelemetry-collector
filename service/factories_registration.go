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

package service

import (
	// This is a temporary workaround to register all factories that are already
	// implemented. This will be removed and factories will be directly registered
	// via code.
	_ "github.com/open-telemetry/opentelemetry-service/exporter/loggingexporter"
	_ "github.com/open-telemetry/opentelemetry-service/exporter/opencensusexporter"
	_ "github.com/open-telemetry/opentelemetry-service/exporter/prometheusexporter"
	_ "github.com/open-telemetry/opentelemetry-service/internal/collector/processor/nodebatcher"
	_ "github.com/open-telemetry/opentelemetry-service/internal/collector/processor/queued"
	_ "github.com/open-telemetry/opentelemetry-service/processor/addattributesprocessor"
	_ "github.com/open-telemetry/opentelemetry-service/receiver/jaegerreceiver"
	_ "github.com/open-telemetry/opentelemetry-service/receiver/opencensusreceiver"
	_ "github.com/open-telemetry/opentelemetry-service/receiver/prometheusreceiver"
	_ "github.com/open-telemetry/opentelemetry-service/receiver/vmmetricsreceiver"
	_ "github.com/open-telemetry/opentelemetry-service/receiver/zipkinreceiver"
)
