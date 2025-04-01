module go.opentelemetry.io/collector/extension/limitermiddlewareextension

go 1.23.0

require (
	github.com/stretchr/testify v1.10.0
	go.opentelemetry.io/collector/component v1.29.0
	go.opentelemetry.io/collector/component/componenttest v0.123.0
	go.opentelemetry.io/collector/config/configlimiter v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/collector/confmap v1.28.1
	go.opentelemetry.io/collector/extension v1.29.0
	go.opentelemetry.io/collector/extension/extensionlimiter v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/collector/extension/extensionmiddleware v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/collector/extension/extensiontest v0.122.1
	go.uber.org/goleak v1.3.0
	google.golang.org/grpc v1.71.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-viper/mapstructure/v2 v2.2.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	github.com/knadh/koanf/maps v0.1.2 // indirect
	github.com/knadh/koanf/providers/confmap v0.1.0 // indirect
	github.com/knadh/koanf/v2 v2.1.2 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/collector/featuregate v1.29.0 // indirect
	go.opentelemetry.io/collector/internal/telemetry v0.123.0 // indirect
	go.opentelemetry.io/collector/pdata v1.29.0 // indirect
	go.opentelemetry.io/contrib/bridges/otelzap v0.10.0 // indirect
	go.opentelemetry.io/otel v1.35.0 // indirect
	go.opentelemetry.io/otel/log v0.11.0 // indirect
	go.opentelemetry.io/otel/metric v1.35.0 // indirect
	go.opentelemetry.io/otel/sdk v1.35.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.35.0 // indirect
	go.opentelemetry.io/otel/trace v1.35.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/net v0.37.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250115164207-1a7da9e5054f // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace go.opentelemetry.io/collector/internal/telemetry => ../../internal/telemetry

replace go.opentelemetry.io/collector/extension/extensionmiddleware => ../../extension/extensionmiddleware

replace go.opentelemetry.io/collector/extension/extensionlimiter => ../../extension/extensionlimiter

replace go.opentelemetry.io/collector/config/configlimiter => ../../config/configlimiter

replace go.opentelemetry.io/collector/pdata => ../../pdata

replace go.opentelemetry.io/collector/featuregate => ../../featuregate

replace go.opentelemetry.io/collector/extension/extensiontest => ../extensiontest

replace go.opentelemetry.io/collector/extension => ../

replace go.opentelemetry.io/collector/confmap => ../../confmap

replace go.opentelemetry.io/collector/pipeline => ../../pipeline

replace go.opentelemetry.io/collector/component/componenttest => ../../component/componenttest

replace go.opentelemetry.io/collector/component => ../../component
