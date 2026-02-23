module go.opentelemetry.io/collector/service/telemetry/telemetrytest

go 1.25.0

require (
	github.com/stretchr/testify v1.11.1
	go.opentelemetry.io/collector/component v1.52.0
	go.opentelemetry.io/collector/pdata v1.52.0
	go.opentelemetry.io/collector/service v0.146.1
	go.opentelemetry.io/otel/metric v1.40.0
	go.opentelemetry.io/otel/trace v1.40.0
	go.uber.org/zap v1.27.1
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.7 // indirect
	github.com/hashicorp/go-version v1.8.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_golang v1.23.2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.67.5 // indirect
	github.com/prometheus/otlptranslator v1.0.0 // indirect
	github.com/prometheus/procfs v0.19.2 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/collector/config/configtelemetry v0.146.1 // indirect
	go.opentelemetry.io/collector/featuregate v1.52.0 // indirect
	go.opentelemetry.io/contrib/otelconf v0.18.0 // indirect
	go.opentelemetry.io/otel v1.40.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc v0.16.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp v0.16.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.40.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.40.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.40.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.40.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.40.0 // indirect
	go.opentelemetry.io/otel/exporters/prometheus v0.60.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdoutlog v0.16.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.40.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.40.0 // indirect
	go.opentelemetry.io/otel/log v0.16.0 // indirect
	go.opentelemetry.io/otel/sdk v1.40.0 // indirect
	go.opentelemetry.io/otel/sdk/log v0.16.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.40.0 // indirect
	go.opentelemetry.io/proto/otlp v1.9.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.yaml.in/yaml/v2 v2.4.3 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/net v0.50.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260128011058-8636f8732409 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260128011058-8636f8732409 // indirect
	google.golang.org/grpc v1.79.1 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
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

replace go.opentelemetry.io/collector/internal/componentalias => ../../../internal/componentalias

replace go.opentelemetry.io/collector/config/confignet => ../../../config/confignet
