module go.opentelemetry.io/collector/processor/processortest

go 1.22.0

require (
	github.com/google/uuid v1.6.0
	github.com/stretchr/testify v1.9.0
	go.opentelemetry.io/collector/component v0.111.0
	go.opentelemetry.io/collector/component/componentstatus v0.111.0
	go.opentelemetry.io/collector/consumer v0.111.1-0.20241021181817-007f06b7c4a8
	go.opentelemetry.io/collector/consumer/consumerprofiles v0.111.0
	go.opentelemetry.io/collector/consumer/consumertest v0.111.0
	go.opentelemetry.io/collector/pdata v1.17.0
	go.opentelemetry.io/collector/pdata/pprofile v0.111.0
	go.opentelemetry.io/collector/pdata/testdata v0.111.0
	go.opentelemetry.io/collector/pipeline v0.111.0
	go.opentelemetry.io/collector/processor v0.111.1-0.20241021181817-007f06b7c4a8
	go.opentelemetry.io/collector/processor/processorprofiles v0.111.0
	go.uber.org/goleak v1.3.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.opentelemetry.io/collector/config/configtelemetry v0.111.0 // indirect
	go.opentelemetry.io/otel v1.31.0 // indirect
	go.opentelemetry.io/otel/metric v1.31.0 // indirect
	go.opentelemetry.io/otel/sdk v1.31.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.31.0 // indirect
	go.opentelemetry.io/otel/trace v1.31.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/net v0.28.0 // indirect
	golang.org/x/sys v0.26.0 // indirect
	golang.org/x/text v0.17.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240822170219-fc7c04adadcd // indirect
	google.golang.org/grpc v1.67.1 // indirect
	google.golang.org/protobuf v1.35.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace go.opentelemetry.io/collector/component => ../../component

replace go.opentelemetry.io/collector/component/componentstatus => ../../component/componentstatus

replace go.opentelemetry.io/collector/consumer => ../../consumer

replace go.opentelemetry.io/collector/consumer/consumerprofiles => ../../consumer/consumerprofiles

replace go.opentelemetry.io/collector/consumer/consumertest => ../../consumer/consumertest

replace go.opentelemetry.io/collector/pdata => ../../pdata

replace go.opentelemetry.io/collector/pdata/pprofile => ../../pdata/pprofile

replace go.opentelemetry.io/collector/pdata/testdata => ../../pdata/testdata

replace go.opentelemetry.io/collector/processor => ../../processor

replace go.opentelemetry.io/collector/processor/processorprofiles => ../../processor/processorprofiles

replace go.opentelemetry.io/collector/config/configtelemetry => ../../config/configtelemetry

replace go.opentelemetry.io/collector/pipeline => ../../pipeline
