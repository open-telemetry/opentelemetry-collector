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
- `npipe`: Windows named pipe configuration (ignored for all other transport types)
  - `security_descriptor`: A [Security Descriptor Definition Language (SDDL)](https://learn.microsoft.com/en-us/windows/win32/secauthz/security-descriptor-definition-language)
    string applied to the named pipe when a listener is created. When empty,
    Windows applies its [default named pipe DACL](https://learn.microsoft.com/en-us/windows/win32/ipc/named-pipe-security-and-access-rights),
    which is roughly equivalent to
    `D:P(A;;GA;;;SY)(A;;GA;;;BA)(A;;0x12019b;;;WD)(A;;0x12019b;;;AN)` — full
    control for `LocalSystem` (`SY`) and `Administrators` (`BA`), and read plus
    limited write for `Everyone` (`WD`) and `Anonymous Logon` (`AN`).

Note that for TCP receivers only the `endpoint` configuration setting is
required.
