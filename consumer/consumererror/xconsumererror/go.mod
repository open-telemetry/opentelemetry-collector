module go.opentelemetry.io/collector/consumer/consumererror/xconsumererror

go 1.24.0

require (
	github.com/stretchr/testify v1.10.0
	go.opentelemetry.io/collector/consumer/consumererror v0.132.0
	go.opentelemetry.io/collector/pdata/pprofile v0.132.0
	go.opentelemetry.io/collector/pdata/testdata v0.132.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.opentelemetry.io/collector/pdata v1.38.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/net v0.40.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.25.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250528174236-200df99c418a // indirect
	google.golang.org/grpc v1.74.2 // indirect
	google.golang.org/protobuf v1.36.7 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace go.opentelemetry.io/collector/pdata => ../../../pdata

replace go.opentelemetry.io/collector/pdata/testdata => ../../../pdata/testdata

replace go.opentelemetry.io/collector/pdata/pprofile => ../../../pdata/pprofile

replace go.opentelemetry.io/collector/consumer/consumererror => ../../consumererror
