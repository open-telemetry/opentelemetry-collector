module go.opentelemetry.io/collector/pdata

go 1.24.0

require (
	github.com/json-iterator/go v1.1.12
	github.com/stretchr/testify v1.11.1
	go.opentelemetry.io/collector/featuregate v1.45.0
	go.opentelemetry.io/proto/slim/otlp v1.9.0
	go.opentelemetry.io/proto/slim/otlp/collector/profiles/v1development v0.2.0
	go.opentelemetry.io/proto/slim/otlp/profiles/v1development v0.2.0
	go.uber.org/goleak v1.3.0
	go.uber.org/multierr v1.11.0
	google.golang.org/grpc v1.76.0
	google.golang.org/protobuf v1.36.10
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/net v0.42.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/text v0.27.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250804133106-a7a43d27e69b // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

retract (
	v1.0.0-rc10 // RC version scheme discovered to be alphabetical, use v1.0.0-rcv0011 instead
	v0.57.1 // Release failed, use v0.57.2
	v0.57.0 // Release failed, use v0.57.2
)

replace go.opentelemetry.io/collector/featuregate => ../featuregate
