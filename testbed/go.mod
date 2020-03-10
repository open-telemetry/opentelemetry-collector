module github.com/open-telemetry/opentelemetry-collector/testbed

go 1.12

require (
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/census-instrumentation/opencensus-proto v0.2.1
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/golang/protobuf v1.3.4
	github.com/open-telemetry/opentelemetry-collector v0.2.4-0.20200115225140-264426a9cae4
	github.com/shirou/gopsutil v2.18.12+incompatible
	github.com/spf13/viper v1.4.1-0.20190911140308-99520c81d86e
	github.com/stretchr/testify v1.5.1
	go.uber.org/zap v1.13.0
)

replace github.com/open-telemetry/opentelemetry-collector => ../

replace github.com/apache/thrift => github.com/apache/thrift v0.0.0-20161221203622-b2a4d4ae21c7

replace github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.4.0
