// package middleware defines the middleware extension for client- and
// server-side RPC interceptors.  HTTP and gRPC-specific interfaces
// are provided using the underyling packages.
//
//	HTTP: net/http
//	gRPC: google.golang.org/grpc
//
// For HTTP clients, the type is http.RoundTripper
// For HTTP servers, the type is http.Handler
//
// For gRPC, there are both unary and stream APIs.
//
// For unary gRPC clients, the type is grpc.UnaryClientInterceptor
// For stream gRPC clients, the type is grpc.StreamClientInterceptor
// For unary gRPC servers, the type is grpc.UnaryServerInterceptor
// For stream gRPC servers, the type is	grpc.StreamServerInterceptor
package extensionmiddleware // import "go.opentelemetry.io/collector/extension/extensionmiddleware"
