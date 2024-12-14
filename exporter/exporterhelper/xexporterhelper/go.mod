module go.opentelemetry.io/collector/exporter/exporterhelper/xexporterhelper

go 1.22.0

require (
	github.com/stretchr/testify v1.10.0
	go.opentelemetry.io/collector/component v0.115.0
	go.opentelemetry.io/collector/component/componenttest v0.115.0
	go.opentelemetry.io/collector/config/configretry v1.21.0
	go.opentelemetry.io/collector/consumer v1.21.0
	go.opentelemetry.io/collector/consumer/consumererror v0.115.0
	go.opentelemetry.io/collector/consumer/consumererror/xconsumererror v0.0.0-20241214150434-e9bc4bde924e
	go.opentelemetry.io/collector/consumer/consumertest v0.115.0
	go.opentelemetry.io/collector/consumer/xconsumer v0.0.0-20241214150434-e9bc4bde924e
	go.opentelemetry.io/collector/exporter v0.115.0
	go.opentelemetry.io/collector/exporter/exportertest v0.115.0
	go.opentelemetry.io/collector/exporter/xexporter v0.0.0-20241214150434-e9bc4bde924e
	go.opentelemetry.io/collector/pdata/pprofile v0.115.0
	go.opentelemetry.io/collector/pdata/testdata v0.115.0
	go.opentelemetry.io/collector/pipeline/xpipeline v0.0.0-20241214150434-e9bc4bde924e
	go.opentelemetry.io/otel v1.32.0
	go.opentelemetry.io/otel/sdk v1.32.0
	go.opentelemetry.io/otel/trace v1.32.0
	go.uber.org/zap v1.27.0
)

require (
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.opentelemetry.io/collector/config/configtelemetry v0.115.0 // indirect
	go.opentelemetry.io/collector/extension v0.115.0 // indirect
	go.opentelemetry.io/collector/extension/experimental/storage v0.115.0 // indirect
	go.opentelemetry.io/collector/featuregate v1.21.0 // indirect
	go.opentelemetry.io/collector/pdata v1.21.0 // indirect
	go.opentelemetry.io/collector/pipeline v0.115.0 // indirect
	go.opentelemetry.io/collector/receiver v0.115.0 // indirect
	go.opentelemetry.io/collector/receiver/receivertest v0.115.0 // indirect
	go.opentelemetry.io/collector/receiver/xreceiver v0.0.0-20241214150434-e9bc4bde924e // indirect
	go.opentelemetry.io/otel/metric v1.32.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.32.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/net v0.29.0 // indirect
	golang.org/x/sys v0.27.0 // indirect
	golang.org/x/text v0.18.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240903143218-8af14fe29dc1 // indirect
	google.golang.org/grpc v1.68.1 // indirect
	google.golang.org/protobuf v1.35.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace go.opentelemetry.io/collector/consumer/consumertest => ../../../consumer/consumertest

replace go.opentelemetry.io/collector/pdata/pprofile => ../../../pdata/pprofile

replace go.opentelemetry.io/collector/pdata/testdata => ../../../pdata/testdata

replace go.opentelemetry.io/collector/exporter => ../../

replace go.opentelemetry.io/collector/consumer => ../../../consumer

replace go.opentelemetry.io/collector/consumer/consumererror/xconsumererror => ../../../consumer/consumererror/xconsumererror

replace go.opentelemetry.io/collector/receiver => ../../../receiver

replace go.opentelemetry.io/collector/consumer/xconsumer => ../../../consumer/xconsumer

replace go.opentelemetry.io/collector/component => ../../../component

replace go.opentelemetry.io/collector/component/componenttest => ../../../component/componenttest

replace go.opentelemetry.io/collector/receiver/xreceiver => ../../../receiver/xreceiver

replace go.opentelemetry.io/collector/receiver/receivertest => ../../../receiver/receivertest

replace go.opentelemetry.io/collector/extension => ../../../extension

replace go.opentelemetry.io/collector/pdata => ../../../pdata

replace go.opentelemetry.io/collector/exporter/xexporter => ../../xexporter

replace go.opentelemetry.io/collector/config/configtelemetry => ../../../config/configtelemetry

replace go.opentelemetry.io/collector/config/configretry => ../../../config/configretry

replace go.opentelemetry.io/collector/pipeline/xpipeline => ../../../pipeline/xpipeline

replace go.opentelemetry.io/collector/extension/experimental/storage => ../../../extension/experimental/storage

replace go.opentelemetry.io/collector/pipeline => ../../../pipeline

replace go.opentelemetry.io/collector/exporter/exportertest => ../../exportertest

replace go.opentelemetry.io/collector/consumer/consumererror => ../../../consumer/consumererror

replace go.opentelemetry.io/collector/extension/extensiontest => ../../../extension/extensiontest

replace go.opentelemetry.io/collector/scraper => ../../../scraper

replace go.opentelemetry.io/collector/featuregate => ../../../featuregate
