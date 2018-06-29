package zipkin

// Tag holds available types
type Tag string

// Common Tag values
const (
	TagHTTPMethod       Tag = "http.method"
	TagHTTPPath         Tag = "http.path"
	TagHTTPUrl          Tag = "http.url"
	TagHTTPStatusCode   Tag = "http.status_code"
	TagHTTPRequestSize  Tag = "http.request.size"
	TagHTTPResponseSize Tag = "http.response.size"
	TagGRPCStatusCode   Tag = "grpc.status_code"
	TagError            Tag = "error"
)

// Set a standard Tag with a payload on provided Span.
func (t Tag) Set(s Span, value string) {
	s.Tag(string(t), value)
}
