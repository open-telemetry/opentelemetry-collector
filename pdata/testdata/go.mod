module go.opentelemetry.io/collector/pdata/testdata

go 1.22.0

require (
	go.opentelemetry.io/collector/pdata v1.22.0
	go.opentelemetry.io/collector/pdata/pprofile v0.116.0
)

require (
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/net v0.33.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241015192408-796eee8c2d53 // indirect
	google.golang.org/grpc v1.69.0 // indirect
	google.golang.org/protobuf v1.36.0 // indirect
)

replace go.opentelemetry.io/collector/pdata => ../

replace go.opentelemetry.io/collector/pdata/pprofile => ../pprofile
