package configauth

import (
	"fmt"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"google.golang.org/grpc/credentials"
	"net/http"
)


// ClientAuth is an Extension that can be used as an authenticator for the configauth.Authentication option.
// Authenticators are then included as part of OpenTelemetry Collector builds and can be referenced by their
// names from the Authentication configuration. .
type ClientAuth interface {
	component.Extension
}

// HTTPClientAuth is a ClientAuth that can be used as an authenticator for the configauth.Authentication option for HTTP
// clients.
type HTTPClientAuth interface {
	ClientAuth
	RoundTripper(base http.RoundTripper) http.RoundTripper
}

// GRPCClientAuth is a ClientAuth that can be used as an authenticator for the configauth.Authentication option for gRPC
// clients.
type GRPCClientAuth interface {
	ClientAuth
	PerRPCCredential() (credentials.PerRPCCredentials, error)
}

// GetHTTPClientAuth attempts to select the appropriate HTTPClientAuth from the list of extensions,
// based on the requested extension name. If an authenticator is not found, an error is returned. This should be only
// used by HTTP clients.
func GetHTTPClientAuth(extensions map[config.NamedEntity]component.Extension, requested string) (HTTPClientAuth, error) {
	if requested == "" {
		return nil, errAuthenticatorNotProvided
	}

	for name, ext := range extensions {
		if name.Name() == requested {
			if auth, ok := ext.(HTTPClientAuth); ok {
				return auth, nil
			} else {
				return nil, fmt.Errorf("requested authenticator is not for HTTP clients")
			}
		}
	}
	return nil, fmt.Errorf("failed to resolve authenticator %q: %w", requested, errAuthenticatorNotFound)
}

// GetGRPCClientAuth attempts to select the appropriate HTTPClientAuth from the list of extensions,
// based on the requested extension name. If an authenticator is not found, an error is returned. This shold only be used
// by gRPC clients
func GetGRPCClientAuth(extensions map[config.NamedEntity]component.Extension, requested string) (GRPCClientAuth, error) {
	if requested == "" {
		return nil, errAuthenticatorNotProvided
	}

	for name, ext := range extensions {
		if name.Name() == requested {
			if auth, ok := ext.(GRPCClientAuth); ok {
				return auth, nil
			} else {
				return nil, fmt.Errorf("requested authenticator is not for gRPC clients")
			}
		}
	}
	return nil, fmt.Errorf("failed to resolve authenticator %q: %w", requested, errAuthenticatorNotFound)
}
