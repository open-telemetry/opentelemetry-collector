// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package builder

const (
	goModTestFile = `// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
module go.opentelemetry.io/collector/cmd/builder/unittests

go 1.21.0

require (
	go.opentelemetry.io/collector/component v0.106.0
	go.opentelemetry.io/collector/confmap v0.106.0
	go.opentelemetry.io/collector/confmap/converter/expandconverter v0.106.0
	go.opentelemetry.io/collector/confmap/provider/envprovider v0.106.0
	go.opentelemetry.io/collector/confmap/provider/fileprovider v0.106.0
	go.opentelemetry.io/collector/confmap/provider/httpprovider v0.106.0
	go.opentelemetry.io/collector/confmap/provider/httpsprovider v0.106.0
	go.opentelemetry.io/collector/confmap/provider/yamlprovider v0.106.0
	go.opentelemetry.io/collector/connector v0.106.0
	go.opentelemetry.io/collector/exporter v0.106.0
	go.opentelemetry.io/collector/exporter/otlpexporter v0.106.0
	go.opentelemetry.io/collector/extension v0.106.0
	go.opentelemetry.io/collector/otelcol v0.106.0
	go.opentelemetry.io/collector/processor v0.106.0
	go.opentelemetry.io/collector/receiver v0.106.0
	go.opentelemetry.io/collector/receiver/otlpreceiver v0.106.0
	golang.org/x/sys v0.20.0
)`

	invalidDependencyGoMod = `// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
module go.opentelemetry.io/collector/cmd/builder/unittests

go 1.21.0

require (
	go.opentelemetry.io/collector/bad/otelcol v0.94.1
	go.opentelemetry.io/collector/component v0.102.1
	go.opentelemetry.io/collector/confmap v0.102.1
	go.opentelemetry.io/collector/confmap/converter/expandconverter v0.102.1
	go.opentelemetry.io/collector/confmap/provider/envprovider v0.102.1
	go.opentelemetry.io/collector/confmap/provider/fileprovider v0.102.1
	go.opentelemetry.io/collector/confmap/provider/httpprovider v0.102.1
	go.opentelemetry.io/collector/confmap/provider/httpsprovider v0.102.1
	go.opentelemetry.io/collector/confmap/provider/yamlprovider v0.102.1
	go.opentelemetry.io/collector/connector v0.102.1
	go.opentelemetry.io/collector/exporter v0.102.1
	go.opentelemetry.io/collector/exporter/otlpexporter v0.102.1
	go.opentelemetry.io/collector/extension v0.102.1
	go.opentelemetry.io/collector/otelcol v0.102.1
	go.opentelemetry.io/collector/processor v0.102.1
	go.opentelemetry.io/collector/receiver v0.102.1
	go.opentelemetry.io/collector/receiver/otlpreceiver v0.102.1
	golang.org/x/sys v0.20.0
)`

	malformedGoMod = `// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
module go.opentelemetry.io/collector/cmd/builder/unittests

go 1.21.0

require (
	go.opentelemetry.io/collector/componentv0.102.1
	go.opentelemetry.io/collector/confmap v0.102.1
	go.opentelemetry.io/collector/confmap/converter/expandconverter v0.102.1
	go.opentelemetry.io/collector/confmap/provider/envprovider v0.102.1
	go.opentelemetry.io/collector/confmap/provider/fileprovider v0.102.1
	go.opentelemetry.io/collector/confmap/provider/httpprovider v0.102.1
	go.opentelemetry.io/collector/confmap/provider/httpsprovider v0.102.1
	go.opentelemetry.io/collector/confmap/provider/yamlprovider v0.102.1
	go.opentelemetry.io/collector/connector v0.102.1
	go.opentelemetry.io/collector/exporter v0.102.1
	go.opentelemetry.io/collector/exporter/otlpexporter v0.102.1
	go.opentelemetry.io/collector/extension v0.102.1
	go.opentelemetry.io/collector/otelcol v0.102.1
	go.opentelemetry.io/collector/processor v0.102.1
	go.opentelemetry.io/collector/receiver v0.102.1
	go.opentelemetry.io/collector/receiver/otlpreceiver v0.102.1
	golang.org/x/sys v0.20.0
)`
)
