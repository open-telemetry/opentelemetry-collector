module go.opentelemetry.io/collector/exporter/otlpexporter

go 1.24.0

require (
	github.com/stretchr/testify v1.11.1
	go.opentelemetry.io/collector v0.142.0
	go.opentelemetry.io/collector/component v1.48.0
	go.opentelemetry.io/collector/component/componenttest v0.142.0
	go.opentelemetry.io/collector/config/configauth v1.48.0
	go.opentelemetry.io/collector/config/configcompression v1.48.0
	go.opentelemetry.io/collector/config/configgrpc v0.142.0
	go.opentelemetry.io/collector/config/configopaque v1.48.0
	go.opentelemetry.io/collector/config/configoptional v1.48.0
	go.opentelemetry.io/collector/config/configretry v1.48.0
	go.opentelemetry.io/collector/config/configtls v1.48.0
	go.opentelemetry.io/collector/confmap v1.48.0
	go.opentelemetry.io/collector/confmap/xconfmap v0.142.0
	go.opentelemetry.io/collector/consumer v1.48.0
	go.opentelemetry.io/collector/consumer/consumererror v0.142.0
	go.opentelemetry.io/collector/exporter v1.48.0
	go.opentelemetry.io/collector/exporter/exporterhelper v0.142.0
	go.opentelemetry.io/collector/exporter/exporterhelper/xexporterhelper v0.142.0
	go.opentelemetry.io/collector/exporter/exportertest v0.142.0
	go.opentelemetry.io/collector/exporter/xexporter v0.142.0
	go.opentelemetry.io/collector/internal/testutil v0.142.0
	go.opentelemetry.io/collector/pdata v1.48.0
	go.opentelemetry.io/collector/pdata/pprofile v0.142.0
	go.opentelemetry.io/collector/pdata/testdata v0.142.0
	go.uber.org/goleak v1.3.0
	go.uber.org/zap v1.27.1
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251029180050-ab9386a59fda
	google.golang.org/grpc v1.78.0
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/foxboron/go-tpm-keyfiles v0.0.0-20250903184740-5d135037bd4d // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/go-tpm v0.9.7 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/go-version v1.8.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/knadh/koanf/maps v0.1.2 // indirect
	github.com/knadh/koanf/providers/confmap v1.0.0 // indirect
	github.com/knadh/koanf/v2 v2.3.0 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/mostynb/go-grpc-compression v1.2.3 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/collector/client v1.48.0 // indirect
	go.opentelemetry.io/collector/config/configmiddleware v1.48.0 // indirect
	go.opentelemetry.io/collector/config/confignet v1.48.0 // indirect
	go.opentelemetry.io/collector/consumer/consumererror/xconsumererror v0.142.0 // indirect
	go.opentelemetry.io/collector/consumer/consumertest v0.142.0 // indirect
	go.opentelemetry.io/collector/consumer/xconsumer v0.142.0 // indirect
	go.opentelemetry.io/collector/extension v1.48.0 // indirect
	go.opentelemetry.io/collector/extension/extensionauth v1.48.0 // indirect
	go.opentelemetry.io/collector/extension/extensionmiddleware v0.142.0 // indirect
	go.opentelemetry.io/collector/extension/xextension v0.142.0 // indirect
	go.opentelemetry.io/collector/featuregate v1.48.0 // indirect
	go.opentelemetry.io/collector/pdata/xpdata v0.142.0 // indirect
	go.opentelemetry.io/collector/pipeline v1.48.0 // indirect
	go.opentelemetry.io/collector/pipeline/xpipeline v0.142.0 // indirect
	go.opentelemetry.io/collector/receiver v1.48.0 // indirect
	go.opentelemetry.io/collector/receiver/receivertest v0.142.0 // indirect
	go.opentelemetry.io/collector/receiver/xreceiver v0.142.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.63.0 // indirect
	go.opentelemetry.io/otel v1.39.0 // indirect
	go.opentelemetry.io/otel/metric v1.39.0 // indirect
	go.opentelemetry.io/otel/sdk v1.39.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.39.0 // indirect
	go.opentelemetry.io/otel/trace v1.39.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.45.0 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace go.opentelemetry.io/collector/component => ../../component

replace go.opentelemetry.io/collector/component/componenttest => ../../component/componenttest

replace go.opentelemetry.io/collector/config/configcompression => ../../config/configcompression

replace go.opentelemetry.io/collector/config/configauth => ../../config/configauth

replace go.opentelemetry.io/collector/config/configgrpc => ../../config/configgrpc

replace go.opentelemetry.io/collector/config/confignet => ../../config/confignet

replace go.opentelemetry.io/collector/config/configopaque => ../../config/configopaque

replace go.opentelemetry.io/collector/config/configtls => ../../config/configtls

replace go.opentelemetry.io/collector/confmap => ../../confmap

replace go.opentelemetry.io/collector/confmap/xconfmap => ../../confmap/xconfmap

replace go.opentelemetry.io/collector/exporter => ../

replace go.opentelemetry.io/collector/extension => ../../extension

replace go.opentelemetry.io/collector/extension/extensionauth => ../../extension/extensionauth

replace go.opentelemetry.io/collector/pdata => ../../pdata

replace go.opentelemetry.io/collector/pdata/testdata => ../../pdata/testdata

replace go.opentelemetry.io/collector/pdata/pprofile => ../../pdata/pprofile

replace go.opentelemetry.io/collector/receiver => ../../receiver

replace go.opentelemetry.io/collector/consumer => ../../consumer

replace go.opentelemetry.io/collector/client => ../../client

replace go.opentelemetry.io/collector/config/configretry => ../../config/configretry

replace go.opentelemetry.io/collector/consumer/xconsumer => ../../consumer/xconsumer

replace go.opentelemetry.io/collector/consumer/consumertest => ../../consumer/consumertest

replace go.opentelemetry.io/collector/receiver/xreceiver => ../../receiver/xreceiver

replace go.opentelemetry.io/collector/receiver/receivertest => ../../receiver/receivertest

replace go.opentelemetry.io/collector/exporter/xexporter => ../xexporter

replace go.opentelemetry.io/collector/pipeline => ../../pipeline

replace go.opentelemetry.io/collector => ../..

replace go.opentelemetry.io/collector/exporter/exporterhelper/xexporterhelper => ../exporterhelper/xexporterhelper

replace go.opentelemetry.io/collector/consumer/consumererror/xconsumererror => ../../consumer/consumererror/xconsumererror

replace go.opentelemetry.io/collector/pipeline/xpipeline => ../../pipeline/xpipeline

replace go.opentelemetry.io/collector/consumer/consumererror => ../../consumer/consumererror

retract (
	v0.76.0 // Depends on retracted pdata v1.0.0-rc10 module, use v0.76.1
	v0.69.0 // Release failed, use v0.69.1
)

replace go.opentelemetry.io/collector/exporter/exportertest => ../exportertest

replace go.opentelemetry.io/collector/extension/extensiontest => ../../extension/extensiontest

replace go.opentelemetry.io/collector/featuregate => ../../featuregate

replace go.opentelemetry.io/collector/extension/xextension => ../../extension/xextension

replace go.opentelemetry.io/collector/extension/extensionauth/extensionauthtest => ../../extension/extensionauth/extensionauthtest

replace go.opentelemetry.io/collector/extension/extensionmiddleware => ../../extension/extensionmiddleware

replace go.opentelemetry.io/collector/config/configmiddleware => ../../config/configmiddleware

replace go.opentelemetry.io/collector/extension/extensionmiddleware/extensionmiddlewaretest => ../../extension/extensionmiddleware/extensionmiddlewaretest

replace go.opentelemetry.io/collector/pdata/xpdata => ../../pdata/xpdata

replace go.opentelemetry.io/collector/config/configoptional => ../../config/configoptional

replace go.opentelemetry.io/collector/exporter/exporterhelper => ../exporterhelper

replace go.opentelemetry.io/collector/internal/testutil => ../../internal/testutil
