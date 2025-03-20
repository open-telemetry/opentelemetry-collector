module go.opentelemetry.io/collector/config/configlimiter

go 1.23.0

replace go.opentelemetry.io/collector/component => ../../component

replace go.opentelemetry.io/collector/extension/extensionmiddleware => ../../extension/extensionmiddleware

//replace go.opentelemetry.io/collector/extension/extensionmiddleware/extensionmiddlewaretest => ../../extension/extensionmiddleware/extensionmiddlewaretest

require (
	go.opentelemetry.io/collector/component v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/collector/extension/extensionmiddleware v0.0.0-00010101000000-000000000000
)

require (
	github.com/gogo/protobuf v1.3.2 // indirect
	go.opentelemetry.io/collector/pdata v1.28.1 // indirect
	go.opentelemetry.io/otel v1.35.0 // indirect
	go.opentelemetry.io/otel/metric v1.35.0 // indirect
	go.opentelemetry.io/otel/trace v1.35.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/net v0.37.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250115164207-1a7da9e5054f // indirect
	google.golang.org/grpc v1.71.0 // indirect
	google.golang.org/protobuf v1.36.5 // indirect
)
