module go.opentelemetry.io/collector/pdata/testdata

go 1.24.0

require (
	go.opentelemetry.io/collector/pdata v1.42.0
	go.opentelemetry.io/collector/pdata/pprofile v0.136.0
)

require (
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	go.opentelemetry.io/collector/featuregate v1.42.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/net v0.41.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250707201910-8d1bb00bc6a7 // indirect
	google.golang.org/grpc v1.75.1 // indirect
	google.golang.org/protobuf v1.36.9 // indirect
)

replace go.opentelemetry.io/collector/pdata => ../

replace go.opentelemetry.io/collector/pdata/pprofile => ../pprofile

replace go.opentelemetry.io/collector/featuregate => ../../featuregate
