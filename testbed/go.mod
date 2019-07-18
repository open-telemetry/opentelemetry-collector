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

replace (
	contrib.go.opencensus.io/exporter/ocagent => github.com/omnition/opencensus-go-exporter-ocagent v0.5.0-gogo
	github.com/census-instrumentation/opencensus-proto => github.com/omnition/opencensus-proto v0.3.0-gogo
	github.com/open-telemetry/opentelemetry-service => ../
	github.com/orijtech/prometheus-go-metrics-exporter => github.com/omnition/prometheus-go-metrics-exporter v0.0.2-gogo
)
