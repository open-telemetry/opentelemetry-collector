module go.opentelemetry.io/collector/config/confignet

go 1.26.0

require (
	github.com/Microsoft/go-winio v0.6.4
	github.com/mdlayher/vsock v1.4.0
	github.com/stretchr/testify v1.12.1
	go.opentelemetry.io/collector/featuregate v1.70.0
	go.uber.org/goleak v1.3.0
)

require (
	github.com/hashicorp/go-version v1.9.0 // indirect
	github.com/mdlayher/socket v0.8.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/net v0.62.0 // indirect
	golang.org/x/sync v0.24.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
)

replace go.opentelemetry.io/collector/featuregate => ../../featuregate
