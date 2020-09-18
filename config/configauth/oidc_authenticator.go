// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package configauth

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc"
	"google.golang.org/grpc"
)

type oidcAuthenticator struct {
	attribute string
	config    OIDC
	provider  *oidc.Provider
	verifier  *oidc.IDTokenVerifier

	unaryInterceptor  unaryInterceptorFunc
	streamInterceptor streamInterceptorFunc
}

var (
	_ Authenticator = (*oidcAuthenticator)(nil)

	errNoClientIDProvided                = errors.New("no ClientID provided for the OIDC configuration")
	errNoIssuerURL                       = errors.New("no IssuerURL provided for the OIDC configuration")
	errInvalidAuthenticationHeaderFormat = errors.New("invalid authorization header format")
	errFailedToObtainClaimsFromToken     = errors.New("failed to get the subject from the token issued by the OIDC provider")
	errClaimNotFound                     = errors.New("username claim from the OIDC configuration not found on the token returned by the OIDC provider")
	errUsernameNotString                 = errors.New("the username returned by the OIDC provider isn't a regular string")
	errGroupsClaimNotFound               = errors.New("groups claim from the OIDC configuration not found on the token returned by the OIDC provider")
	errNotAuthenticated                  = errors.New("authentication didn't succeed")
)

func newOIDCAuthenticator(cfg Authentication) (*oidcAuthenticator, error) {
	if cfg.OIDC.Audience == "" {
		return nil, errNoClientIDProvided
	}
	if cfg.OIDC.IssuerURL == "" {
		return nil, errNoIssuerURL
	}
	if cfg.Attribute == "" {
		cfg.Attribute = defaultAttribute
	}

	return &oidcAuthenticator{
		attribute:         cfg.Attribute,
		config:            *cfg.OIDC,
		unaryInterceptor:  defaultUnaryInterceptor,
		streamInterceptor: defaultStreamInterceptor,
	}, nil
}

func (o *oidcAuthenticator) Authenticate(ctx context.Context, headers map[string][]string) (context.Context, error) {
	authHeaders := headers[o.attribute]
	if len(authHeaders) == 0 {
		return ctx, errNotAuthenticated
	}

	// we only use the first header, if multiple values exist
	parts := strings.Split(authHeaders[0], " ")
	if len(parts) != 2 {
		return ctx, errInvalidAuthenticationHeaderFormat
	}

	idToken, err := o.verifier.Verify(ctx, parts[1])
	if err != nil {
		return ctx, fmt.Errorf("failed to verify token: %w", err)
	}

	claims := map[string]interface{}{}
	if err = idToken.Claims(&claims); err != nil {
		// currently, this isn't a valid condition, the Verify call a few lines above
		// will already attempt to parse the payload as a json and set it as the claims
		// for the token. As we are using a map to hold the claims, there's no way to fail
		// to read the claims. It could fail if we were using a custom struct. Instead of
		// swalling the error, it's better to make this future-proof, in case the underlying
		// code changes
		return ctx, errFailedToObtainClaimsFromToken
	}

	sub, err := getSubjectFromClaims(claims, o.config.UsernameClaim, idToken.Subject)
	if err != nil {
		return ctx, fmt.Errorf("failed to get subject from claims in the token: %w", err)
	}
	ctx = context.WithValue(ctx, subjectKey, sub)

	gr, err := getGroupsFromClaims(claims, o.config.GroupsClaim)
	if err != nil {
		return ctx, fmt.Errorf("failed to get groups from claims in the token: %w", err)
	}
	ctx = context.WithValue(ctx, groupsKey, gr)

	return ctx, nil
}

func (o *oidcAuthenticator) Start(context.Context) error {
	provider, err := getProviderForConfig(o.config)
	if err != nil {
		return fmt.Errorf("failed to get configuration from the auth server: %w", err)
	}
	o.provider = provider

	o.verifier = o.provider.Verifier(&oidc.Config{
		ClientID: o.config.Audience,
	})

	return nil
}

func (o *oidcAuthenticator) Close() error {
	// no-op at the moment
	// once we implement caching of the tokens we might need this
	return nil
}

func (o *oidcAuthenticator) UnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return o.unaryInterceptor(ctx, req, info, handler, o.Authenticate)
}

func (o *oidcAuthenticator) StreamInterceptor(srv interface{}, str grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	return o.streamInterceptor(srv, str, info, handler, o.Authenticate)
}

func getSubjectFromClaims(claims map[string]interface{}, usernameClaim string, fallback string) (string, error) {
	if len(usernameClaim) > 0 {
		username, found := claims[usernameClaim]
		if !found {
			return "", errClaimNotFound
		}

		sUsername, ok := username.(string)
		if !ok {
			return "", errUsernameNotString
		}

		return sUsername, nil
	}

	return fallback, nil
}

func getGroupsFromClaims(claims map[string]interface{}, groupsClaim string) ([]string, error) {
	if len(groupsClaim) > 0 {
		var groups []string
		rawGroup, ok := claims[groupsClaim]
		if !ok {
			return nil, errGroupsClaimNotFound
		}
		switch v := rawGroup.(type) {
		case string:
			groups = append(groups, v)
		case []string:
			groups = v
		case []interface{}:
			groups = make([]string, 0, len(v))
			for i := range v {
				groups = append(groups, fmt.Sprintf("%v", v[i]))
			}
		}

		return groups, nil
	}

	return []string{}, nil
}

func getProviderForConfig(config OIDC) (*oidc.Provider, error) {
	t := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 10 * time.Second,
			DualStack: true,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	cert, err := getIssuerCACertFromPath(config.IssuerCAPath)
	if err != nil {
		return nil, err // the errors from this path have enough context already
	}

	if cert != nil {
		t.TLSClientConfig = &tls.Config{
			RootCAs: x509.NewCertPool(),
		}
		t.TLSClientConfig.RootCAs.AddCert(cert)
	}

	client := &http.Client{
		Timeout:   5 * time.Second,
		Transport: t,
	}
	oidcContext := oidc.ClientContext(context.Background(), client)
	return oidc.NewProvider(oidcContext, config.IssuerURL)
}

func getIssuerCACertFromPath(path string) (*x509.Certificate, error) {
	if path == "" {
		return nil, nil
	}

	rawCA, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read the CA file %q: %w", path, err)
	}

	if len(rawCA) == 0 {
		return nil, fmt.Errorf("could not read the CA file %q: empty file", path)
	}

	block, _ := pem.Decode(rawCA)
	if block == nil {
		return nil, fmt.Errorf("cannot decode the contents of the CA file %q: %w", path, err)
	}

	return x509.ParseCertificate(block.Bytes)
}
