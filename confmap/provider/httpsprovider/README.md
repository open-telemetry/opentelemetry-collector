### What is the httpsprovider?

An implementation of `confmap.Provider` for HTTPS (httpsprovider) allows OTEL Collector to use the HTTPS protocol to
load configuration files stored in web servers.

Expected URI format:
- https://...

### Prerequistes

You need to setup a HTTP server with support to HTTPS. The server must have a certificate that can be validated in the
host running the collector using system root certificates.

### Configuration

At this moment, this component only support communicating with servers whose certificate can be verified using the root
CA certificates installed in the system. The process of adding more root CA certificates to the system is operating
system dependent. For Linux, please refer to the `update-ca-trust` command.
