module go.opentelemetry.io/collector/config/configtls

go 1.24

require (
	github.com/foxboron/go-tpm-keyfiles v0.0.0-20250323135004-b31fac66206e
	github.com/fsnotify/fsnotify v1.9.0
	github.com/google/go-tpm v0.9.5
	github.com/stretchr/testify v1.11.1
	go.opentelemetry.io/collector/config/configopaque v1.41.0
	go.opentelemetry.io/collector/confmap/xconfmap v0.135.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/google/go-tpm-tools v0.4.4 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	github.com/knadh/koanf/maps v0.1.2 // indirect
	github.com/knadh/koanf/providers/confmap v1.0.0 // indirect
	github.com/knadh/koanf/v2 v2.2.2 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.opentelemetry.io/collector/confmap v1.41.0 // indirect
	go.opentelemetry.io/collector/featuregate v1.41.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.35.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace go.opentelemetry.io/collector/config/configopaque => ../configopaque

replace go.opentelemetry.io/collector/featuregate => ../../featuregate

replace go.opentelemetry.io/collector/confmap => ../../confmap

replace go.opentelemetry.io/collector/confmap/xconfmap => ../../confmap/xconfmap
