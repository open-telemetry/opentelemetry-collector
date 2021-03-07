module go.opentelemetry.io/collector/cmd/schemagen

go 1.15

require (
	github.com/fatih/structtag v1.2.0
	github.com/stretchr/testify v1.7.0
  github.com/mostynb/go-grpc-compression v1.1.6
	go.opentelemetry.io/collector v0.0.0-00010101000000-000000000000
	gopkg.in/yaml.v2 v2.4.0
)

replace go.opentelemetry.io/collector => ../../
