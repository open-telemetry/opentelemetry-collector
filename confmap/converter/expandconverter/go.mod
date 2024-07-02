module go.opentelemetry.io/collector/confmap/converter/expandconverter

go 1.21.0

require (
	go.opentelemetry.io/collector v0.104.0
	go.opentelemetry.io/collector/confmap v0.104.0
	go.uber.org/goleak v1.3.0
	go.uber.org/zap v1.27.0
)

require (
	github.com/go-viper/mapstructure/v2 v2.0.0-alpha.1 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	github.com/knadh/koanf/maps v0.1.1 // indirect
	github.com/knadh/koanf/providers/confmap v0.1.0 // indirect
	github.com/knadh/koanf/v2 v2.1.1 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	go.opentelemetry.io/collector/featuregate v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace go.opentelemetry.io/collector/component => ../../../component

replace go.opentelemetry.io/collector/confmap => ../..

replace go.opentelemetry.io/collector => ../../..

replace go.opentelemetry.io/collector/config/configtelemetry => ../../../config/configtelemetry

replace go.opentelemetry.io/collector/pdata/testdata => ../../../pdata/testdata

replace go.opentelemetry.io/collector/pdata => ../../../pdata

replace go.opentelemetry.io/collector/featuregate => ../../../featuregate

replace go.opentelemetry.io/collector/consumer => ../../../consumer

replace go.opentelemetry.io/collector/pdata/pprofile => ../../../pdata/pprofile
