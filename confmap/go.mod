module go.opentelemetry.io/collector/confmap

go 1.24.0

require (
	github.com/go-viper/mapstructure/v2 v2.4.0
	github.com/gobwas/glob v0.2.3
	github.com/knadh/koanf/maps v0.1.2
	github.com/knadh/koanf/providers/confmap v1.0.0
	github.com/knadh/koanf/v2 v2.3.0
	github.com/stretchr/testify v1.11.1
	go.opentelemetry.io/collector/featuregate v1.48.0
	go.opentelemetry.io/collector/internal/testutil v0.142.0
	go.uber.org/goleak v1.3.0
	go.uber.org/multierr v1.11.0
	go.uber.org/zap v1.27.1
	go.yaml.in/yaml/v3 v3.0.4
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/hashicorp/go-version v1.8.0 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

retract (
	v0.76.0 // Depends on retracted pdata v1.0.0-rc10 module, use v0.76.1
	v0.69.0 // Release failed, use v0.69.1
)

replace (
	go.opentelemetry.io/collector/featuregate => ../featuregate
	go.opentelemetry.io/collector/internal/testutil => ../internal/testutil
)
