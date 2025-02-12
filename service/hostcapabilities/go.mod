module go.opentelemetry.io/collector/service/hostcapabilities

go 1.23.0

require go.opentelemetry.io/collector/service v0.119.0

require (
	github.com/gogo/protobuf v1.3.2 // indirect
	go.opentelemetry.io/collector/component v0.120.0 // indirect
	go.opentelemetry.io/collector/pdata v1.26.0 // indirect
	go.opentelemetry.io/otel v1.34.0 // indirect
	go.opentelemetry.io/otel/metric v1.34.0 // indirect
	go.opentelemetry.io/otel/trace v1.34.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/net v0.35.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
	golang.org/x/text v0.22.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250115164207-1a7da9e5054f // indirect
	google.golang.org/grpc v1.70.0 // indirect
	google.golang.org/protobuf v1.36.5 // indirect
)

replace (
	go.opentelemetry.io/collector/component => ../../component
	go.opentelemetry.io/collector/component/componentstatus => ../../component/componentstatus
	go.opentelemetry.io/collector/component/componenttest => ../../component/componenttest
	go.opentelemetry.io/collector/config/configtelemetry => ../../config/configtelemetry
	go.opentelemetry.io/collector/confmap => ../../confmap
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
	go.opentelemetry.io/collector/extension/extensioncapabilities => ../../extension/extensioncapabilities
	go.opentelemetry.io/collector/extension/extensiontest => ../../extension/extensiontest
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
