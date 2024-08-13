module go.opentelemetry.io/collector/consumer/consumertest

go 1.21.0

replace go.opentelemetry.io/collector/consumer => ../

require (
	github.com/stretchr/testify v1.9.0
	go.opentelemetry.io/collector/consumer v0.107.0
	go.opentelemetry.io/collector/consumer/consumerprofiles v0.107.0
	go.opentelemetry.io/collector/pdata v1.13.0
	go.opentelemetry.io/collector/pdata/pprofile v0.107.0
	go.opentelemetry.io/collector/pdata/testdata v0.107.0
	go.uber.org/goleak v1.3.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/net v0.25.0 // indirect
	golang.org/x/sys v0.20.0 // indirect
	golang.org/x/text v0.15.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240528184218-531527333157 // indirect
	google.golang.org/grpc v1.65.0 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace go.opentelemetry.io/collector/pdata/pprofile => ../../pdata/pprofile

replace go.opentelemetry.io/collector/pdata => ../../pdata

replace go.opentelemetry.io/collector/consumer/consumerprofiles => ../consumerprofiles

replace go.opentelemetry.io/collector/pdata/testdata => ../../pdata/testdata
