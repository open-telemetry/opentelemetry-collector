module github.com/open-telemetry/opentelemetry-service/testbed

go 1.12

require (
	contrib.go.opencensus.io/exporter/jaeger v0.1.1-0.20190430175949-e8b55949d948
	github.com/open-telemetry/opentelemetry-service v0.0.0-20190625135304-4bd705a25a35
	github.com/shirou/gopsutil v2.18.12+incompatible
	github.com/spf13/viper v1.4.0
	github.com/stretchr/testify v1.3.0
	go.opencensus.io v0.22.0
)

replace github.com/open-telemetry/opentelemetry-service => ../
