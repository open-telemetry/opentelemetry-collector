package configauth

import (
	"fmt"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"google.golang.org/grpc/credentials"
	"net/http"
)


type ClientAuth interface {
	component.Extension
}

type HTTPClientAuth interface {
	ClientAuth
	RoundTripper(base http.RoundTripper) http.RoundTripper
}

type GRPCClientAuth interface {
	ClientAuth
	PerRPCCredential()(credentials.PerRPCCredentials, error)
}


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


func GetGRPCClientAuth(extensions map[config.NamedEntity]component.Extension, requested string) (GRPCClientAuth, error) {
		return nil, fmt.Errorf("not implemented ")
}
