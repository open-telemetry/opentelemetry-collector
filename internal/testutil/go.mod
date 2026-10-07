module go.opentelemetry.io/collector/internal/testutil

go 1.26.0

require (
	github.com/stretchr/testify v1.12.1
	go.uber.org/goleak v1.3.0
)

require (
	github.com/hashicorp/go-version v1.9.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
)

require (
	go.opentelemetry.io/collector/featuregate v1.67.0
	go.yaml.in/yaml/v3 v3.0.5 // indirect
)

replace go.opentelemetry.io/collector/featuregate => ../../featuregate
