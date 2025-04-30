# TLS Configuration Settings

Crypto TLS exposes a [variety of settings](https://godoc.org/crypto/tls).
Several of these settings are available for configuration within individual
receivers or exporters.

Note that mutual TLS (mTLS) is also supported.

## TLS / mTLS Configuration

By default, TLS is enabled:

- `insecure` (default = false): whether to enable client transport security for
  the exporter's HTTPs or gRPC connection. See
  [grpc.WithInsecure()](https://godoc.org/google.golang.org/grpc#WithInsecure)
  for gRPC.
- `curve_preferences` (default = []): specify your curve preferences  that will
	 be used in an ECDHE handshake, in preference order. Accepted values are:
  - X25519
  - P521
  - P256
  - P384

As a result, the following parameters are also required:

- `cert_file`: Path to the TLS cert to use for TLS required connections. Should
  only be used if `insecure` is set to false.
  - `cert_pem`: Alternative to `cert_file`. Provide the certificate contents as a string instead of a filepath.

- `key_file`: Path to the TLS key to use for TLS required connections. Should
  only be used if `insecure` is set to false.
  - `key_pem`: Alternative to `key_file`. Provide the key contents as a string instead of a filepath.

A certificate authority may also need to be defined:

- `ca_file`: Path to the CA cert. For a client this verifies the server
  certificate. For a server this verifies client certificates. If empty uses
  system root CA. Should only be used if `insecure` is set to false.
  - `ca_pem`: Alternative to `ca_file`. Provide the CA cert contents as a string instead of a filepath.

You can also combine defining a certificate authority with the system certificate authorities.

- `include_system_ca_certs_pool` (default = false): whether to load the system certificate authorities pool
  alongside the certificate authority.

Additionally you can configure TLS to be enabled but skip verifying the server's
certificate chain. This cannot be combined with `insecure` since `insecure`
won't use TLS at all.

- `insecure_skip_verify` (default = false): whether to skip verifying the
  certificate or not.

Minimum and maximum TLS version can be set:

__IMPORTANT__: TLS 1.0 and 1.1 are deprecated due to known vulnerabilities and should be avoided.

- `min_version` (default = "1.2"): Minimum acceptable TLS version.
  - options: ["1.0", "1.1", "1.2", "1.3"]

- `max_version` (default = "" handled by [crypto/tls](https://github.com/golang/go/blob/ed9db1d36ad6ef61095d5941ad9ee6da7ab6d05a/src/crypto/tls/common.go#L700) - currently TLS 1.3): Maximum acceptable TLS version.
  - options: ["1.0", "1.1", "1.2", "1.3"]

Explicit cipher suites can be set. If left blank, a safe default list is used. See https://go.dev/src/crypto/tls/cipher_suites.go for a list of supported cipher suites.
- `cipher_suites`: (default = []): List of cipher suites to use.

Example:
```
  cipher_suites:
    - TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256
    - TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
    - TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256
```

Additionally certificates may be reloaded by setting the below configuration.

- `reload_interval` (optional) : ReloadInterval specifies the duration after which the certificate will be reloaded.
   If not set, it will never be reloaded.
   Accepts a [duration string](https://pkg.go.dev/time#ParseDuration),
   valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".

How TLS/mTLS is configured depends on whether configuring the client or server.
See below for examples.

- `tpm` (optional): Use the trusted platform module to retrieve the TLS key.

## Client Configuration

[Exporters](https://github.com/open-telemetry/opentelemetry-collector/blob/main/exporter/README.md)
leverage client configuration. The TLS configuration parameters are defined
under `tls`, like server configuration.

Beyond TLS configuration, the following setting can optionally be configured:

- `server_name_override`: If set to a non-empty string, it will override the
  virtual host name of authority (e.g. :authority header field) in requests
  (typically used for testing).

Example:

```yaml
exporters:
  otlp:
    endpoint: myserver.local:55690
    tls:
      insecure: false
      ca_file: server.crt
      cert_file: client.crt
      key_file: client.key
      min_version: "1.1"
      max_version: "1.2"
  otlp/insecure:
    endpoint: myserver.local:55690
    tls:
      insecure: true
  otlp/secure_no_verify:
    endpoint: myserver.local:55690
    tls:
      insecure: false
      insecure_skip_verify: true
```

## Server Configuration

[Receivers](https://github.com/open-telemetry/opentelemetry-collector/blob/main/receiver/README.md)
leverage server configuration.

Beyond TLS configuration, the following setting can optionally be configured
(required for mTLS):

- `client_ca_file`: Path to the TLS cert to use by the server to verify a
  client certificate. (optional) This sets the ClientCAs and ClientAuth to
  RequireAndVerifyClientCert in the TLSConfig. Please refer to
  https://godoc.org/crypto/tls#Config for more information.
- `client_ca_file_reload` (default = false): Reload the ClientCAs file when it is modified.

Example:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: mysite.local:55690
        tls:
          cert_file: server.crt
          key_file: server.key
  otlp/mtls:
    protocols:
      grpc:
        endpoint: mysite.local:55690
        tls:
          client_ca_file: client.pem
          cert_file: server.crt
          key_file: server.key
  otlp/notls:
    protocols:
      grpc:
        endpoint: mysite.local:55690
```

## Trusted platform module (TPM) configuration

The [trusted platform module](https://trustedcomputinggroup.org/resource/trusted-platform-module-tpm-summary/) (TPM) configuration can be used for loading TLS key from TPM. Currently only TSS2 format is supported.

- `enabled` (default = false): Enables loading `tls.key_file` from TPM.

- `path` (default = ""): The path to the TPM device or Unix domain socket. For instance `/dev/tpm0` or `/dev/tpmrm0`. This option is not supported on Windows.

- `owner_auth` (default = ""): The owner authorization value. This is used to authenticate the TPM device. If not set, the default owner authorization will be used.

- `auth` (default = ""): The authorization value. This is used to authenticate the TPM device. If not set, the default authorization will be used.

Example:

```yaml
exporters:
  otlp:
    endpoint: myserver.local:55690
    tls:
      ca_file: ca.crt
      cert_file: client.crt
      key_file: client-tss2.key
      tpm:
        enabled: true
        path: /dev/tpmrm0
```

The `client-tss2.key` private key with TSS2 format will be loaded from the TPM device `/dev/tpmrm0`.