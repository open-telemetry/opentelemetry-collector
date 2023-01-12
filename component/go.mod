module go.opentelemetry.io/collector/component

go 1.18

require (
	github.com/stretchr/testify v1.8.1
	go.opentelemetry.io/collector v0.69.1
	go.opentelemetry.io/collector/confmap v0.69.1
	go.opentelemetry.io/otel/metric v0.34.0
	go.opentelemetry.io/otel/trace v1.11.2
	go.uber.org/multierr v1.9.0
	go.uber.org/zap v1.24.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/knadh/koanf v1.4.4 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.opentelemetry.io/collector/featuregate v0.69.1 // indirect
	go.opentelemetry.io/otel v1.11.2 // indirect
	go.uber.org/atomic v1.10.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace go.opentelemetry.io/collector => ../

replace go.opentelemetry.io/collector/confmap => ../confmap

replace go.opentelemetry.io/collector/consumer => ../consumer

replace go.opentelemetry.io/collector/featuregate => ../featuregate

replace go.opentelemetry.io/collector/pdata => ../pdata

replace go.opentelemetry.io/collector/semconv => ../semconv

replace go.opentelemetry.io/collector/extension/zpagesextension => ../extension/zpagesextension
