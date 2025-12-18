module go.opentelemetry.io/collector/config/configtls

go 1.24.0

require (
	github.com/foxboron/go-tpm-keyfiles v0.0.0-20250903184740-5d135037bd4d
	github.com/fsnotify/fsnotify v1.9.0
	github.com/google/go-tpm v0.9.7
	github.com/stretchr/testify v1.11.1
	go.opentelemetry.io/collector/config/configopaque v1.48.0
	go.opentelemetry.io/collector/confmap/xconfmap v0.142.0
	go.opentelemetry.io/collector/internal/testutil v0.142.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/google/go-tpm-tools v0.4.4 // indirect
	github.com/hashicorp/go-version v1.8.0 // indirect
	github.com/knadh/koanf/maps v0.1.2 // indirect
	github.com/knadh/koanf/providers/confmap v1.0.0 // indirect
	github.com/knadh/koanf/v2 v2.3.0 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.opentelemetry.io/collector/confmap v1.48.0 // indirect
	go.opentelemetry.io/collector/featuregate v1.48.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.1 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.45.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace go.opentelemetry.io/collector/config/configopaque => ../configopaque

replace go.opentelemetry.io/collector/featuregate => ../../featuregate

replace go.opentelemetry.io/collector/confmap => ../../confmap

replace go.opentelemetry.io/collector/confmap/xconfmap => ../../confmap/xconfmap

replace go.opentelemetry.io/collector/internal/testutil => ../../internal/testutil
