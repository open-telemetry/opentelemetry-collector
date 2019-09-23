module github.com/open-telemetry/opentelemetry-service/testbed

go 1.12

require (
	contrib.go.opencensus.io/exporter/jaeger v0.1.1-0.20190430175949-e8b55949d948
	github.com/open-telemetry/opentelemetry-service v0.0.0-20190625135304-4bd705a25a35
	github.com/shirou/gopsutil v2.18.12+incompatible
	github.com/shirou/w32 v0.0.0-20160930032740-bb4de0191aa4 // indirect
	github.com/spf13/viper v1.4.1-0.20190911140308-99520c81d86e
	github.com/stretchr/testify v1.4.0
	go.opencensus.io v0.22.1
)

replace github.com/open-telemetry/opentelemetry-service => ../
