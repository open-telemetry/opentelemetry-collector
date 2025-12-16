module go.opentelemetry.io/collector/service/hostcapabilities

go 1.24.0

require (
	go.opentelemetry.io/collector/component v1.48.0
	go.opentelemetry.io/collector/pipeline v1.48.0
	go.opentelemetry.io/collector/service v0.142.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/hashicorp/go-version v1.8.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	go.opentelemetry.io/collector/featuregate v1.48.0 // indirect
	go.opentelemetry.io/collector/pdata v1.48.0 // indirect
	go.opentelemetry.io/otel v1.39.0 // indirect
	go.opentelemetry.io/otel/metric v1.39.0 // indirect
	go.opentelemetry.io/otel/trace v1.39.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.1 // indirect
)

replace (
	go.opentelemetry.io/collector/client => ../../client
	go.opentelemetry.io/collector/component => ../../component
	go.opentelemetry.io/collector/component/componentstatus => ../../component/componentstatus
	go.opentelemetry.io/collector/component/componenttest => ../../component/componenttest
	go.opentelemetry.io/collector/config/configauth => ../../config/configauth
	go.opentelemetry.io/collector/config/configcompression => ../../config/configcompression
	go.opentelemetry.io/collector/config/confighttp => ../../config/confighttp
	go.opentelemetry.io/collector/config/configopaque => ../../config/configopaque
	go.opentelemetry.io/collector/config/configretry => ../../config/configretry
	go.opentelemetry.io/collector/config/configtelemetry => ../../config/configtelemetry
	go.opentelemetry.io/collector/config/configtls => ../../config/configtls
	go.opentelemetry.io/collector/confmap => ../../confmap
	go.opentelemetry.io/collector/confmap/xconfmap => ../../confmap/xconfmap
	go.opentelemetry.io/collector/connector => ../../connector
	go.opentelemetry.io/collector/connector/connectortest => ../../connector/connectortest
	go.opentelemetry.io/collector/connector/xconnector => ../../connector/xconnector
	go.opentelemetry.io/collector/consumer => ../../consumer
	go.opentelemetry.io/collector/consumer/consumererror => ../../consumer/consumererror
	go.opentelemetry.io/collector/consumer/consumertest => ../../consumer/consumertest
	go.opentelemetry.io/collector/consumer/xconsumer => ../../consumer/xconsumer
	go.opentelemetry.io/collector/exporter => ../../exporter
	go.opentelemetry.io/collector/exporter/exportertest => ../../exporter/exportertest
	go.opentelemetry.io/collector/exporter/xexporter => ../../exporter/xexporter
	go.opentelemetry.io/collector/extension => ../../extension
	go.opentelemetry.io/collector/extension/extensionauth => ../../extension/extensionauth
	go.opentelemetry.io/collector/extension/extensionauth/extensionauthtest => ../../extension/extensionauth/extensionauthtest
	go.opentelemetry.io/collector/extension/extensioncapabilities => ../../extension/extensioncapabilities
	go.opentelemetry.io/collector/extension/extensiontest => ../../extension/extensiontest
	go.opentelemetry.io/collector/extension/xextension => ../../extension/xextension
	go.opentelemetry.io/collector/extension/zpagesextension => ../../extension/zpagesextension
	go.opentelemetry.io/collector/featuregate => ../../featuregate
	go.opentelemetry.io/collector/internal/fanoutconsumer => ../../internal/fanoutconsumer
	go.opentelemetry.io/collector/internal/telemetry => ../../internal/telemetry
	go.opentelemetry.io/collector/pdata => ../../pdata
	go.opentelemetry.io/collector/pdata/pprofile => ../../pdata/pprofile
	go.opentelemetry.io/collector/pdata/testdata => ../../pdata/testdata
	go.opentelemetry.io/collector/pipeline => ../../pipeline
	go.opentelemetry.io/collector/pipeline/xpipeline => ../../pipeline/xpipeline
	go.opentelemetry.io/collector/processor => ../../processor
	go.opentelemetry.io/collector/processor/processortest => ../../processor/processortest
	go.opentelemetry.io/collector/processor/xprocessor => ../../processor/xprocessor
	go.opentelemetry.io/collector/receiver => ../../receiver
	go.opentelemetry.io/collector/receiver/receivertest => ../../receiver/receivertest
	go.opentelemetry.io/collector/receiver/xreceiver => ../../receiver/xreceiver

	go.opentelemetry.io/collector/service => ..
)

replace go.opentelemetry.io/collector/otelcol => ../../otelcol

replace go.opentelemetry.io/collector/confmap/provider/fileprovider => ../../confmap/provider/fileprovider

replace go.opentelemetry.io/collector/extension/extensionmiddleware/extensionmiddlewaretest => ../../extension/extensionmiddleware/extensionmiddlewaretest

replace go.opentelemetry.io/collector/config/configmiddleware => ../../config/configmiddleware

replace go.opentelemetry.io/collector/extension/extensionmiddleware => ../../extension/extensionmiddleware

replace go.opentelemetry.io/collector/pdata/xpdata => ../../pdata/xpdata

replace go.opentelemetry.io/collector/exporter/exporterhelper => ../../exporter/exporterhelper

replace go.opentelemetry.io/collector/config/configoptional => ../../config/configoptional

replace go.opentelemetry.io/collector/service/telemetry/telemetrytest => ../telemetry/telemetrytest

replace go.opentelemetry.io/collector/internal/testutil => ../../internal/testutil
