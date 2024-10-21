module go.opentelemetry.io/collector

// NOTE:
// This go.mod is NOT used to build any official binary.
// To see the builder manifests used for official binaries,
// check https://github.com/open-telemetry/opentelemetry-collector-releases
//
// For the OpenTelemetry Collector Core distribution specifically, see
// https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol

go 1.22.0

require (
	github.com/stretchr/testify v1.9.0
	go.opentelemetry.io/collector/component v0.111.0
	go.opentelemetry.io/collector/component/componentstatus v0.111.0
	go.opentelemetry.io/collector/consumer v0.111.1-0.20241021181817-007f06b7c4a8
	go.opentelemetry.io/collector/consumer/consumerprofiles v0.111.0
	go.opentelemetry.io/collector/consumer/consumertest v0.111.0
	go.opentelemetry.io/collector/pdata v1.17.0
	go.opentelemetry.io/collector/pdata/pprofile v0.111.0
	go.opentelemetry.io/collector/pdata/testdata v0.111.0
	go.uber.org/goleak v1.3.0
	go.uber.org/multierr v1.11.0
	google.golang.org/grpc v1.67.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.12.0 // indirect
	go.opentelemetry.io/collector/config/configtelemetry v0.111.0 // indirect
	go.opentelemetry.io/collector/pipeline v0.111.0 // indirect
	go.opentelemetry.io/otel v1.31.0 // indirect
	go.opentelemetry.io/otel/metric v1.31.0 // indirect
	go.opentelemetry.io/otel/sdk v1.31.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.31.0 // indirect
	go.opentelemetry.io/otel/trace v1.31.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/net v0.28.0 // indirect
	golang.org/x/sys v0.26.0 // indirect
	golang.org/x/text v0.18.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240822170219-fc7c04adadcd // indirect
	google.golang.org/protobuf v1.35.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace go.opentelemetry.io/collector/component => ./component

replace go.opentelemetry.io/collector/component/componentstatus => ./component/componentstatus

replace go.opentelemetry.io/collector/config/configtelemetry => ./config/configtelemetry

replace go.opentelemetry.io/collector/consumer => ./consumer

replace go.opentelemetry.io/collector/consumer/consumertest => ./consumer/consumertest

replace go.opentelemetry.io/collector/pdata => ./pdata

replace go.opentelemetry.io/collector/pdata/testdata => ./pdata/testdata

retract (
	v0.76.0 // Depends on retracted pdata v1.0.0-rc10 module, use v0.76.1
	v0.69.0 // Release failed, use v0.69.1
	v0.57.1 // Release failed, use v0.57.2
	v0.57.0 // Release failed, use v0.57.2
	v0.32.0 // Contains incomplete metrics transition to proto 0.9.0, random components are not working.
)

replace go.opentelemetry.io/collector/pdata/pprofile => ./pdata/pprofile

replace go.opentelemetry.io/collector/consumer/consumerprofiles => ./consumer/consumerprofiles

replace go.opentelemetry.io/collector/pipeline => ./pipeline
