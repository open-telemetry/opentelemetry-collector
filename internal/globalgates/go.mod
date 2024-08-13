module go.opentelemetry.io/collector/internal/globalgates

go 1.21.0

require go.opentelemetry.io/collector/featuregate v1.13.0

require (
	github.com/hashicorp/go-version v1.7.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
)

replace go.opentelemetry.io/collector/featuregate => ../../featuregate
