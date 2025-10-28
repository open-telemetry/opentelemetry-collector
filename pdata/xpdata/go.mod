module go.opentelemetry.io/collector/pdata/xpdata

go 1.24.0

require (
	github.com/stretchr/testify v1.11.1
	go.opentelemetry.io/collector/client v1.44.0
	go.opentelemetry.io/collector/featuregate v1.44.0
	go.opentelemetry.io/collector/pdata v1.44.0
	go.opentelemetry.io/collector/pdata/pprofile v0.138.0
	go.opentelemetry.io/collector/pdata/testdata v0.138.0
	go.opentelemetry.io/otel/trace v1.38.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.opentelemetry.io/otel v1.38.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace go.opentelemetry.io/collector/pdata => ..

replace go.opentelemetry.io/collector/pdata/pprofile => ../pprofile

replace go.opentelemetry.io/collector/pdata/testdata => ../testdata

replace go.opentelemetry.io/collector/client => ../../client

replace go.opentelemetry.io/collector/consumer => ../../consumer

replace go.opentelemetry.io/collector/featuregate => ../../featuregate
