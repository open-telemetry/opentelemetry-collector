# Threat model

The [Security Response Guidelines](https://github.com/open-telemetry/sig-security/blob/main/security-response.md)
describe the general OpenTelemetry security model and vulnerability reporting
process. This document adds context specific to the Collector. Developer
guidance is in [Security best practices](docs/security-best-practices.md);
deployment guidance is in the [Collector configuration](https://opentelemetry.io/docs/security/config-best-practices/)
and [hosting](https://opentelemetry.io/docs/security/hosting-best-practices/)
best practices.

## Scope and assets

This repository provides the Collector service, shared libraries, components,
and build tools. A distribution selects its components and configuration; its
attack surface depends on what it includes and enables.

Assets include:

- Telemetry received, processed, buffered, and exported by the Collector.
- Credentials, certificates, configuration, and other data accessible to the
  Collector process.
- The availability of telemetry pipelines and the host resources they share.
- Source, dependencies, build credentials, and published artifacts used to
  build and distribute the Collector.

## Actors and trust boundaries

### Operators and configuration authors

Operators select the distribution, components, configuration sources, network
exposure, destinations, credentials, and process permissions. Configuration is
trusted input: it can intentionally read local files and environment variables,
load configuration from network sources, and select exporter destinations.
See [configuration providers](confmap/README.md#provider).

An actor authorized to supply the full Collector configuration can exercise
these capabilities with the Collector's permissions. Merely demonstrating a
configured file read or outbound request does not establish a vulnerability.
A report must identify authority beyond that intentionally granted, such as
telemetry influencing a protected configuration source or bypassing an enforced
destination restriction.

Operators must protect configuration sources and credentials, restrict exposed
endpoints, and use appropriate operating system permissions. A less trusted
actor gaining access to these resources through a Collector defect is a
different case from an operator intentionally granting access.

### Component and distribution authors

Components and configuration providers execute in the Collector process with
its permissions. They are trusted code; the Collector does not sandbox a
malicious component. Separate pipelines in one process do not provide process
or resource isolation. Deployments requiring isolation between mutually
untrusted operators or components need an external isolation boundary.

These assumptions do not excuse bypasses of specific authentication,
authorization, or data separation controls that a component promises to enforce.

### Telemetry producers and network peers

Receivers accept telemetry from producers. Depending on the enabled components,
the Collector also reads remote scrape targets, exporter responses, and other
network input. Authentication identifies an immediate peer; it does not prove
the origin or trustworthiness of every field the peer forwards. A producer can,
for example, record an application caller's input as a telemetry attribute.

Trace IDs, resource attributes, log bodies, and client-supplied metadata are not
inherently authenticated tenant identities. A deployment that uses them for
routing or data separation must establish their trust through appropriate
authentication and authorization. See [batching and client metadata](processor/batchprocessor/README.md#batching-and-client-metadata).

Parsing and forwarding telemetry containing a URL, path, query, or markup does
not by itself establish an injection vulnerability. Identify an operation that
interprets that value with unintended authority. Likewise, a configured remote
destination does not make all data returned by that destination trusted.

### Local and diagnostic access

The Collector runs with the permissions of its service account. It does not
provide a boundary against an actor already able to fully control its
executable, effective configuration, or process memory. An actor with more
limited local access may still cross a boundary through a Collector defect.

Internal telemetry and diagnostic endpoints can expose operational details.
Operators must restrict their access independently of receiver access; enabling
authentication on a telemetry receiver does not protect every endpoint.

The [debug exporter](exporter/debugexporter/README.md) intentionally outputs
telemetry, including full records at detailed verbosity. Operators are
responsible for selecting data to collect and export. Unexpected collection
of credentials or failure of a configured redaction control still requires
security assessment, as does failure to mask sensitive configuration fields
as described in the developer security guidance.

### Build and release inputs

Contributions, dependencies, and generated inputs can reach build and release
tooling. Assess whether untrusted content can gain execution, credentials,
signing, or publication authority. A production configuration trust assumption
does not make an untrusted pull request safe to execute in a privileged workflow.

## Availability

The Collector follows the [OpenTelemetry security model](https://github.com/open-telemetry/sig-security/blob/main/security-response.md#security-model).
Default settings are not guaranteed to be secure against availability attacks,
and denial of service or other availability attacks by properly authenticated
clients are not considered security vulnerabilities. These failures can still
be reported and fixed as ordinary bugs.

Authenticated producers share the Collector process and its finite resources.
The Collector does not promise availability isolation between these clients.
The exclusion keeps context-dependent reliability failures from automatically
triggering vulnerability handling, inflated severity ratings, and security
advisories that burden users and divert attention from other vulnerabilities.
It does not imply that these failures are unimportant or should remain unfixed.

This exclusion applies to availability-only failures, including crashes caused
by properly authenticated telemetry submissions. Authentication does not make
every forwarded attribute trustworthy, as described above, but the presence of
caller-controlled values alone does not remove the authenticated-client
availability exclusion. Authentication or authorization bypasses, unintended
disclosure, and integrity failures still require security assessment.

Operators are responsible for protecting telemetry endpoints with authentication
and appropriate network restrictions, and for configuring resource protections
for their deployment. Receiver authentication does not imply that an exposed
diagnostic endpoint is also protected.

The Collector does not guarantee lossless delivery. Documented backpressure or
data drops, such as memory-limiter rejection, queue overflow, and exhaustion of
configured retries during a backend outage, are expected capacity or reliability
behavior.

Outside the exclusions above, an availability report must identify a concrete
security risk: how attacker-controlled input exploits a reachable processing or
retention weakness to deny service, beyond ordinary traffic exceeding deployment
capacity. A missing resource limit alone does not establish a vulnerability.
Changes made solely to reduce resource consumption or improve performance are
capacity or hardening work. Security mitigations should be limited to what is
necessary to address the identified risk.

For example, an unauthenticated caller bypassing a configured decompressed
request-size limit and exhausting process memory warrants security assessment.
Increasing queue capacity to accommodate more legitimate traffic is capacity
work.

Evaluate the actual protections and their scope:

- [HTTP](config/confighttp/README.md) and [gRPC](config/configgrpc/README.md)
  server limits apply to particular request paths. Identify whether the
  expensive work happens before a relevant limit, including decompression.
- The [memory limiter](processor/memorylimiterprocessor/README.md) checks
  memory periodically and refuses telemetry in a pipeline. Incoming data can
  allocate memory before reaching it; it is not a bound on all receiver work.
- [Sending queues](exporter/exporterhelper/README.md#sending-queue) and
  [batch processors](processor/batchprocessor/README.md) have different
  sizing and retention settings. A request or item count is not a byte limit,
  and `send_batch_size` triggers sending rather than enforcing a maximum size.
  Record the relevant queue, batch, and metadata cardinality settings.

Availability evidence should include input size, rate, concurrency, resource
cost, and how long resources remain occupied. Describe affected pipelines and
users, recovery requirements, and why applicable protections are ineffective.
Distinguish expected rejection or telemetry loss from a stalled pipeline,
Collector process failure, or effects on other services. Assess component
compatibility assumptions, such as those inherited from Prometheus, where
relevant, rather than assuming every endpoint has the same threat model.

## Reporting and assessment

Use the private reporting process in the Security Response Guidelines for
suspected vulnerabilities. Include the following where known; maintainers may
request missing details:

1. The affected version or commit, component or workflow, and relevant
   distribution and configuration.
1. Who controls the input, the exact values they control, and required access
   or privileges, including authentication to intermediate systems.
1. The reachable path from that input to the affected operation and the
   protection or trust boundary that is missing or bypassed.
1. Concrete confidentiality, integrity, or availability consequences, whose
   data or service is affected, and a reproducer or supporting code evidence.
1. Existing protections and why they do not prevent the demonstrated impact.

A dependency advisory, scanner warning, or hypothetical crash alone is not
evidence of an exploitable Collector vulnerability. Automated reports need the
same reachability and impact evidence as other reports and should distinguish
verified behavior from assumptions. Severity must reflect demonstrated attack
prerequisites and impact; the presence of a crash alone does not establish a
high or critical rating. Acceptance, severity, and response prioritization
follow the Security Response Guidelines.
