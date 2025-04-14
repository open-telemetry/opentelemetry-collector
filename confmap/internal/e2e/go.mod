module go.opentelemetry.io/collector/confmap/internal/e2e

go 1.23.0

require (
	github.com/stretchr/testify v1.10.0
	go.opentelemetry.io/collector/config/configopaque v1.30.0
	go.opentelemetry.io/collector/confmap v1.30.0
	go.opentelemetry.io/collector/confmap/provider/envprovider v1.30.0
	go.opentelemetry.io/collector/confmap/provider/fileprovider v1.30.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-viper/mapstructure/v2 v2.2.1 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	github.com/knadh/koanf/maps v0.1.2 // indirect
	github.com/knadh/koanf/providers/confmap v0.1.0 // indirect
	github.com/knadh/koanf/v2 v2.1.2 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.opentelemetry.io/collector/featuregate v1.30.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	sigs.k8s.io/yaml v1.4.0 // indirect
)

replace go.opentelemetry.io/collector/confmap => ../../

replace go.opentelemetry.io/collector/confmap/provider/fileprovider => ../../provider/fileprovider

replace go.opentelemetry.io/collector/confmap/provider/envprovider => ../../provider/envprovider

replace go.opentelemetry.io/collector/config/configopaque => ../../../config/configopaque

replace go.opentelemetry.io/collector/featuregate => ../../../featuregate
