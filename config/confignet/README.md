# Network Configuration Settings

[Receivers](https://github.com/open-telemetry/opentelemetry-collector/blob/main/receiver/README.md)
leverage network configuration to set connection and transport information.

- `endpoint`: Configures the address for this network connection. For TCP and
  UDP networks, the address has the form "host:port". The host must be a
  literal IP address, or a host name that can be resolved to IP addresses. The
  port must be a literal port number or a service name. If the host is a
  literal IPv6 address it must be enclosed in square brackets, as in
  "[2001:db8::1]:80" or "[fe80::1%zone]:80". The zone specifies the scope of
  the literal IPv6 address as defined in RFC 4007.
- `transport`: Known protocols are "tcp", "tcp4" (IPv4-only), "tcp6"
  (IPv6-only), "udp", "udp4" (IPv4-only), "udp6" (IPv6-only), "ip", "ip4"
  (IPv4-only), "ip6" (IPv6-only), "unix", "unixgram", "unixpacket" and
  "npipe" (Windows named pipes, Windows-only).
- `dialer`: Dialer configuration
  - `timeout`: Dialer timeout is the maximum amount of time a dial will wait for a connect to complete. The default is no timeout.
- `socket_permissions`: File permissions applied to a filesystem-based Unix
  domain socket file after binding. Only applies to the `unix`, `unixgram`
  and `unixpacket` transports; ignored for abstract sockets (endpoints
  starting with `@`) and all other transports. Defaults to `0722`, which
  allows any local process to connect to the socket while only the owner
  can otherwise manage it.

Note that for TCP receivers only the `endpoint` configuration setting is
required.
