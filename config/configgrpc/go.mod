module go.opentelemetry.io/collector/config/configgrpc

go 1.23.0

require (
	github.com/mostynb/go-grpc-compression v1.2.3
	github.com/stretchr/testify v1.10.0
	go.opentelemetry.io/collector/client v1.36.0
	go.opentelemetry.io/collector/component v1.36.0
	go.opentelemetry.io/collector/component/componenttest v0.130.0
	go.opentelemetry.io/collector/config/configauth v0.130.0
	go.opentelemetry.io/collector/config/configcompression v1.36.0
	go.opentelemetry.io/collector/config/configmiddleware v0.130.0
	go.opentelemetry.io/collector/config/confignet v1.36.0
	go.opentelemetry.io/collector/config/configopaque v1.36.0
	go.opentelemetry.io/collector/config/configoptional v0.130.0
	go.opentelemetry.io/collector/config/configtls v1.36.0
	go.opentelemetry.io/collector/extension v1.36.0
	go.opentelemetry.io/collector/extension/extensionauth v1.36.0
	go.opentelemetry.io/collector/extension/extensionauth/extensionauthtest v0.130.0
	go.opentelemetry.io/collector/extension/extensionmiddleware v0.130.0
	go.opentelemetry.io/collector/extension/extensionmiddleware/extensionmiddlewaretest v0.130.0
	go.opentelemetry.io/collector/pdata v1.36.0
	go.opentelemetry.io/collector/pdata/testdata v0.130.0
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.62.0
	go.opentelemetry.io/otel v1.37.0
	go.uber.org/goleak v1.3.0
	google.golang.org/grpc v1.73.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/foxboron/go-tpm-keyfiles v0.0.0-20250323135004-b31fac66206e // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-viper/mapstructure/v2 v2.3.0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/go-tpm v0.9.5 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/knadh/koanf/maps v0.1.2 // indirect
	github.com/knadh/koanf/providers/confmap v1.0.0 // indirect
	github.com/knadh/koanf/v2 v2.2.1 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/collector/confmap v1.36.0 // indirect
	go.opentelemetry.io/collector/featuregate v1.36.0 // indirect
	go.opentelemetry.io/collector/internal/telemetry v0.130.0 // indirect
	go.opentelemetry.io/collector/pdata/pprofile v0.130.0 // indirect
	go.opentelemetry.io/contrib/bridges/otelzap v0.12.0 // indirect
	go.opentelemetry.io/otel/log v0.13.0 // indirect
	go.opentelemetry.io/otel/metric v1.37.0 // indirect
	go.opentelemetry.io/otel/sdk v1.37.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.37.0 // indirect
	go.opentelemetry.io/otel/trace v1.37.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.39.0 // indirect
	golang.org/x/net v0.41.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250603155806-513f23925822 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace go.opentelemetry.io/collector/client => ../../client

replace go.opentelemetry.io/collector/config/configauth => ../configauth

replace go.opentelemetry.io/collector/config/configcompression => ../configcompression

replace go.opentelemetry.io/collector/config/confignet => ../confignet

replace go.opentelemetry.io/collector/config/configopaque => ../configopaque

replace go.opentelemetry.io/collector/config/configoptional => ../configoptional

replace go.opentelemetry.io/collector/config/configtls => ../configtls

replace go.opentelemetry.io/collector/extension => ../../extension

replace go.opentelemetry.io/collector/extension/extensionauth => ../../extension/extensionauth

replace go.opentelemetry.io/collector/config/configmiddleware => ../configmiddleware

replace go.opentelemetry.io/collector/extension/extensionmiddleware => ../../extension/extensionmiddleware

replace go.opentelemetry.io/collector/pdata => ../../pdata

replace go.opentelemetry.io/collector/pdata/testdata => ../../pdata/testdata

replace go.opentelemetry.io/collector/pdata/pprofile => ../../pdata/pprofile

replace go.opentelemetry.io/collector/component => ../../component

replace go.opentelemetry.io/collector/component/componenttest => ../../component/componenttest

replace go.opentelemetry.io/collector/consumer => ../../consumer

replace go.opentelemetry.io/collector/extension/extensionauth/extensionauthtest => ../../extension/extensionauth/extensionauthtest

replace go.opentelemetry.io/collector/internal/telemetry => ../../internal/telemetry

replace go.opentelemetry.io/collector/pipeline => ../../pipeline

replace go.opentelemetry.io/collector/featuregate => ../../featuregate

replace go.opentelemetry.io/collector/extension/extensionmiddleware/extensionmiddlewaretest => ../../extension/extensionmiddleware/extensionmiddlewaretest

replace go.opentelemetry.io/collector/confmap => ../../confmap
