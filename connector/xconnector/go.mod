module go.opentelemetry.io/collector/connector/xconnector

go 1.22.0

require (
	github.com/stretchr/testify v1.10.0
	go.opentelemetry.io/collector/component v0.118.0
	go.opentelemetry.io/collector/connector v0.118.0
	go.opentelemetry.io/collector/consumer v1.24.0
	go.opentelemetry.io/collector/consumer/consumertest v0.118.0
	go.opentelemetry.io/collector/consumer/xconsumer v0.118.0
	go.opentelemetry.io/collector/internal/fanoutconsumer v0.118.0
	go.opentelemetry.io/collector/pdata/pprofile v0.118.0
	go.opentelemetry.io/collector/pdata/testdata v0.118.0
	go.opentelemetry.io/collector/pipeline v0.118.0
	go.opentelemetry.io/collector/pipeline/xpipeline v0.118.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.opentelemetry.io/collector/config/configtelemetry v0.118.0 // indirect
	go.opentelemetry.io/collector/pdata v1.24.0 // indirect
	go.opentelemetry.io/otel v1.34.0 // indirect
	go.opentelemetry.io/otel/metric v1.34.0 // indirect
	go.opentelemetry.io/otel/trace v1.34.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/net v0.33.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241015192408-796eee8c2d53 // indirect
	google.golang.org/grpc v1.69.4 // indirect
	google.golang.org/protobuf v1.36.4 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace go.opentelemetry.io/collector/connector => ../

replace go.opentelemetry.io/collector/consumer => ../../consumer

replace go.opentelemetry.io/collector/consumer/consumertest => ../../consumer/consumertest

replace go.opentelemetry.io/collector/pdata/pprofile => ../../pdata/pprofile

replace go.opentelemetry.io/collector/consumer/xconsumer => ../../consumer/xconsumer

replace go.opentelemetry.io/collector/component => ../../component

replace go.opentelemetry.io/collector/pdata => ../../pdata

replace go.opentelemetry.io/collector/config/configtelemetry => ../../config/configtelemetry

replace go.opentelemetry.io/collector/pdata/testdata => ../../pdata/testdata

replace go.opentelemetry.io/collector/pipeline => ../../pipeline

replace go.opentelemetry.io/collector/pipeline/xpipeline => ../../pipeline/xpipeline

replace go.opentelemetry.io/collector/internal/fanoutconsumer => ../../internal/fanoutconsumer
