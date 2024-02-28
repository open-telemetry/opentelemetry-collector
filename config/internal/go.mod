module go.opentelemetry.io/collector/config/internal

go 1.21

require (
	github.com/stretchr/testify v1.8.4
	go.opentelemetry.io/collector v0.95.0
	go.uber.org/goleak v1.3.0
	go.uber.org/zap v1.26.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/hashicorp/go-version v1.6.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.opentelemetry.io/collector/featuregate v1.2.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace go.opentelemetry.io/collector => ../../

replace go.opentelemetry.io/collector/featuregate => ../../featuregate

replace go.opentelemetry.io/collector/confmap => ../../confmap

replace go.opentelemetry.io/collector/config/configtelemetry => ../configtelemetry

replace go.opentelemetry.io/collector/pdata => ../../pdata

replace go.opentelemetry.io/collector/consumer => ../../consumer

replace go.opentelemetry.io/collector/component => ../../component
