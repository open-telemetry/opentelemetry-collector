module go.opentelemetry.io/collector/service/telemetry/telemetrytest

go 1.24.0

require (
	github.com/stretchr/testify v1.11.1
	go.opentelemetry.io/collector/component v1.47.0
	go.opentelemetry.io/collector/pdata v1.47.0
	go.opentelemetry.io/collector/service v0.141.0
	go.opentelemetry.io/otel/metric v1.39.0
	go.opentelemetry.io/otel/trace v1.39.0
	go.uber.org/zap v1.27.1
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/hashicorp/go-version v1.8.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	go.opentelemetry.io/collector/config/configtelemetry v0.141.0 // indirect
	go.opentelemetry.io/collector/featuregate v1.47.0 // indirect
	go.opentelemetry.io/contrib/otelconf v0.18.0 // indirect
	go.opentelemetry.io/otel v1.39.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace go.opentelemetry.io/collector/component => ../../../component

replace go.opentelemetry.io/collector/internal/telemetry => ../../../internal/telemetry/

replace go.opentelemetry.io/collector/pdata => ../../../pdata

replace go.opentelemetry.io/collector/confmap => ../../../confmap

replace go.opentelemetry.io/collector/config/configtelemetry => ../../../config/configtelemetry

replace go.opentelemetry.io/collector/featuregate => ../../../featuregate

replace go.opentelemetry.io/collector/service => ../../

replace go.opentelemetry.io/collector/extension/extensionmiddleware => ../../../extension/extensionmiddleware

replace go.opentelemetry.io/collector/config/configmiddleware => ../../../config/configmiddleware

replace go.opentelemetry.io/collector/receiver => ../../../receiver

replace go.opentelemetry.io/collector/processor/processortest => ../../../processor/processortest

replace go.opentelemetry.io/collector/processor/xprocessor => ../../../processor/xprocessor

replace go.opentelemetry.io/collector/processor => ../../../processor

replace go.opentelemetry.io/collector/pdata/xpdata => ../../../pdata/xpdata

replace go.opentelemetry.io/collector/pdata/pprofile => ../../../pdata/pprofile

replace go.opentelemetry.io/collector/extension/extensiontest => ../../../extension/extensiontest

replace go.opentelemetry.io/collector/exporter/xexporter => ../../../exporter/xexporter

replace go.opentelemetry.io/collector/consumer/consumertest => ../../../consumer/consumertest

replace go.opentelemetry.io/collector/pipeline => ../../../pipeline

replace go.opentelemetry.io/collector/receiver/xreceiver => ../../../receiver/xreceiver

replace go.opentelemetry.io/collector/pipeline/xpipeline => ../../../pipeline/xpipeline

replace go.opentelemetry.io/collector/pdata/testdata => ../../../pdata/testdata

replace go.opentelemetry.io/collector/otelcol => ../../../otelcol

replace go.opentelemetry.io/collector/extension/extensioncapabilities => ../../../extension/extensioncapabilities

replace go.opentelemetry.io/collector/extension/xextension => ../../../extension/xextension

replace go.opentelemetry.io/collector/exporter => ../../../exporter

replace go.opentelemetry.io/collector/config/configoptional => ../../../config/configoptional

replace go.opentelemetry.io/collector/extension/extensionmiddleware/extensionmiddlewaretest => ../../../extension/extensionmiddleware/extensionmiddlewaretest

replace go.opentelemetry.io/collector/config/configcompression => ../../../config/configcompression

replace go.opentelemetry.io/collector/extension => ../../../extension

replace go.opentelemetry.io/collector/exporter/exporterhelper => ../../../exporter/exporterhelper

replace go.opentelemetry.io/collector/consumer => ../../../consumer

replace go.opentelemetry.io/collector/consumer/xconsumer => ../../../consumer/xconsumer

replace go.opentelemetry.io/collector/consumer/consumererror => ../../../consumer/consumererror

replace go.opentelemetry.io/collector/connector => ../../../connector

replace go.opentelemetry.io/collector/confmap/xconfmap => ../../../confmap/xconfmap

replace go.opentelemetry.io/collector/config/configtls => ../../../config/configtls

replace go.opentelemetry.io/collector/service/hostcapabilities => ../../hostcapabilities

replace go.opentelemetry.io/collector/extension/zpagesextension => ../../../extension/zpagesextension

replace go.opentelemetry.io/collector/config/configretry => ../../../config/configretry

replace go.opentelemetry.io/collector/config/confighttp => ../../../config/confighttp

replace go.opentelemetry.io/collector/config/configopaque => ../../../config/configopaque

replace go.opentelemetry.io/collector/config/configauth => ../../../config/configauth

replace go.opentelemetry.io/collector/extension/extensionauth/extensionauthtest => ../../../extension/extensionauth/extensionauthtest

replace go.opentelemetry.io/collector/client => ../../../client

replace go.opentelemetry.io/collector/receiver/receivertest => ../../../receiver/receivertest

replace go.opentelemetry.io/collector/confmap/provider/fileprovider => ../../../confmap/provider/fileprovider

replace go.opentelemetry.io/collector/internal/fanoutconsumer => ../../../internal/fanoutconsumer

replace go.opentelemetry.io/collector/exporter/exportertest => ../../../exporter/exportertest

replace go.opentelemetry.io/collector/extension/extensionauth => ../../../extension/extensionauth

replace go.opentelemetry.io/collector/connector/xconnector => ../../../connector/xconnector

replace go.opentelemetry.io/collector/connector/connectortest => ../../../connector/connectortest

replace go.opentelemetry.io/collector/component/componenttest => ../../../component/componenttest

replace go.opentelemetry.io/collector/component/componentstatus => ../../../component/componentstatus

replace go.opentelemetry.io/collector/internal/testutil => ../../../internal/testutil
