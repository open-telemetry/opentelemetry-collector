module go.opentelemetry.io/collector/config/configmiddleware

go 1.24.0

require (
	github.com/stretchr/testify v1.11.1
	go.opentelemetry.io/collector/component v1.48.0
	go.opentelemetry.io/collector/extension v1.48.0
	go.opentelemetry.io/collector/extension/extensionmiddleware v0.142.0
	go.opentelemetry.io/collector/extension/extensionmiddleware/extensionmiddlewaretest v0.142.0
	google.golang.org/grpc v1.78.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/hashicorp/go-version v1.8.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.opentelemetry.io/collector/featuregate v1.48.0 // indirect
	go.opentelemetry.io/collector/pdata v1.48.0 // indirect
	go.opentelemetry.io/otel v1.39.0 // indirect
	go.opentelemetry.io/otel/metric v1.39.0 // indirect
	go.opentelemetry.io/otel/trace v1.39.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.1 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251029180050-ab9386a59fda // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace go.opentelemetry.io/collector/component => ../../component

replace go.opentelemetry.io/collector/extension/extensionmiddleware => ../../extension/extensionmiddleware

replace go.opentelemetry.io/collector/extension/extensionmiddleware/extensionmiddlewaretest => ../../extension/extensionmiddleware/extensionmiddlewaretest

replace go.opentelemetry.io/collector/pdata => ../../pdata

replace go.opentelemetry.io/collector/featuregate => ../../featuregate

replace go.opentelemetry.io/collector/extension => ../../extension

replace go.opentelemetry.io/collector/internal/testutil => ../../internal/testutil
