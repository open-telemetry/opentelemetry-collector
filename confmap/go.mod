module go.opentelemetry.io/collector/confmap

go 1.21

require (
	github.com/go-viper/mapstructure/v2 v2.0.0-alpha.1
	github.com/knadh/koanf/maps v0.1.1
	github.com/knadh/koanf/providers/confmap v0.1.0
	github.com/knadh/koanf/v2 v2.1.0
	github.com/stretchr/testify v1.8.4
	go.uber.org/goleak v1.3.0
	go.uber.org/multierr v1.11.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
)

retract (
	v0.76.0 // Depends on retracted pdata v1.0.0-rc10 module, use v0.76.1
	v0.69.0 // Release failed, use v0.69.1
)
