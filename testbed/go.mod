module github.com/open-telemetry/opentelemetry-collector/testbed

go 1.12

require (
	contrib.go.opencensus.io/exporter/jaeger v0.1.1-0.20190430175949-e8b55949d948
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/open-telemetry/opentelemetry-collector v0.2.1-0.20191205212659-419ed61e5bac
	github.com/shirou/gopsutil v2.18.12+incompatible
	github.com/spf13/viper v1.4.1-0.20190911140308-99520c81d86e
	github.com/stretchr/testify v1.4.0
	go.opencensus.io v0.22.1
	go.uber.org/zap v1.10.0
)

replace github.com/open-telemetry/opentelemetry-collector => ../
