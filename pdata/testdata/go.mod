module go.opentelemetry.io/collector/pdata/testdata

go 1.23.0

require (
	go.opentelemetry.io/collector/pdata v1.30.0
	go.opentelemetry.io/collector/pdata/pprofile v0.123.0
)

require (
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/net v0.37.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250115164207-1a7da9e5054f // indirect
	google.golang.org/grpc v1.71.1 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
)

replace go.opentelemetry.io/collector/pdata => ../

replace go.opentelemetry.io/collector/pdata/pprofile => ../pprofile
