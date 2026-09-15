module go.opentelemetry.io/collector/featuregate

go 1.26.0

require (
	github.com/hashicorp/go-version v1.9.0
	github.com/stretchr/testify v1.12.1
	go.uber.org/goleak v1.3.0
	go.uber.org/multierr v1.11.0
)

require go.yaml.in/yaml/v3 v3.0.5 // indirect

retract (
	v0.76.0 // Depends on retracted pdata v1.0.0-rc10 module, use v0.76.1
	v0.69.0 // Release failed, use v0.69.1
)
