module go.opentelemetry.io/collector/config/confighttp/xconfighttp

go 1.25.0

require (
	github.com/stretchr/testify v1.11.1
	go.opentelemetry.io/collector/component/componenttest v0.146.1
	go.opentelemetry.io/collector/config/confighttp v0.146.1
	go.opentelemetry.io/collector/config/confignet v1.52.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.65.0
	go.opentelemetry.io/otel/sdk v1.40.0
	go.opentelemetry.io/otel/trace v1.40.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/foxboron/go-tpm-keyfiles v0.0.0-20251226215517-609e4778396f // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/google/go-tpm v0.9.8 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/go-version v1.8.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.18.4 // indirect
	github.com/knadh/koanf/maps v0.1.2 // indirect
	github.com/knadh/koanf/providers/confmap v1.0.0 // indirect
	github.com/knadh/koanf/v2 v2.3.2 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/pierrec/lz4/v4 v4.1.25 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rs/cors v1.11.1 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/collector/client v1.52.0 // indirect
	go.opentelemetry.io/collector/component v1.52.0 // indirect
	go.opentelemetry.io/collector/config/configauth v1.52.0 // indirect
	go.opentelemetry.io/collector/config/configcompression v1.52.0 // indirect
	go.opentelemetry.io/collector/config/configmiddleware v1.52.0 // indirect
	go.opentelemetry.io/collector/config/configopaque v1.52.0 // indirect
	go.opentelemetry.io/collector/config/configoptional v1.52.0 // indirect
	go.opentelemetry.io/collector/config/configtls v1.52.0 // indirect
	go.opentelemetry.io/collector/confmap v1.52.0 // indirect
	go.opentelemetry.io/collector/confmap/xconfmap v0.146.1 // indirect
	go.opentelemetry.io/collector/extension/extensionauth v1.52.0 // indirect
	go.opentelemetry.io/collector/extension/extensionmiddleware v0.146.1 // indirect
	go.opentelemetry.io/collector/featuregate v1.52.0 // indirect
	go.opentelemetry.io/collector/pdata v1.52.0 // indirect
	go.opentelemetry.io/otel v1.40.0 // indirect
	go.opentelemetry.io/otel/metric v1.40.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.40.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.1 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.48.0 // indirect
	golang.org/x/net v0.50.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251222181119-0a764e51fe1b // indirect
	google.golang.org/grpc v1.79.1 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace go.opentelemetry.io/collector/config/confighttp => ../../confighttp

replace go.opentelemetry.io/collector/client => ../../../client

replace go.opentelemetry.io/collector/consumer => ../../../consumer

replace go.opentelemetry.io/collector/extension/extensionauth/extensionauthtest => ../../../extension/extensionauth/extensionauthtest

replace go.opentelemetry.io/collector/component/componenttest => ../../../component/componenttest

replace go.opentelemetry.io/collector/config/configauth => ../../configauth

replace go.opentelemetry.io/collector/config/configoptional => ../../configoptional

replace go.opentelemetry.io/collector/pdata => ../../../pdata

replace go.opentelemetry.io/collector/extension/extensionauth => ../../../extension/extensionauth

replace go.opentelemetry.io/collector/config/configopaque => ../../configopaque

replace go.opentelemetry.io/collector/component => ../../../component

replace go.opentelemetry.io/collector/extension => ../../../extension

replace go.opentelemetry.io/collector/config/configtls => ../../configtls

replace go.opentelemetry.io/collector/config/configcompression => ../../configcompression

replace go.opentelemetry.io/collector/featuregate => ../../../featuregate

replace go.opentelemetry.io/collector/extension/extensionmiddleware => ../../../extension/extensionmiddleware

replace go.opentelemetry.io/collector/config/configmiddleware => ../../configmiddleware

replace go.opentelemetry.io/collector/config/confignet => ../../confignet

replace go.opentelemetry.io/collector/extension/extensionmiddleware/extensionmiddlewaretest => ../../../extension/extensionmiddleware/extensionmiddlewaretest

replace go.opentelemetry.io/collector/confmap => ../../../confmap

replace go.opentelemetry.io/collector/confmap/xconfmap => ../../../confmap/xconfmap

replace go.opentelemetry.io/collector/internal/testutil => ../../../internal/testutil

replace go.opentelemetry.io/collector/internal/componentalias => ../../../internal/componentalias
