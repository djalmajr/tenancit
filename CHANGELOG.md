# Changelog

All notable changes will be documented in this file. The project follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and will adopt semantic
versioning when the first public release is tagged.

## [Unreleased]

## [0.1.2] - 2026-07-28

### Fixed

- Preserve the resource display name in admin listings — the row mappers
  dropped `DisplayName` when building `ResourceHeader`.
- Finish the delete-dialog i18n split (`deleteInputLabelPrefix`/`Suffix`
  around the styled slug).

### Added

- Portable chart: `ingress.enabled`/`ingress.className`/`ingress.tlsSecretName`
  (existing TLS secret suppresses the cert-manager annotation) and
  `adminAuth.mode` (`oidc` default; `legacy_shared_token` for ClusterIP-only
  bootstrap via port-forward). Default and personal renders stay byte-identical.

## [0.1.1] - 2026-07-12

### Security

- Reject out-of-range Valkey counters and durations before narrowing or
  converting them.
- Parse API usage page limits directly at the database integer width.
- Compute idempotency fingerprints incrementally without overflow-prone
  allocation sizing.

## [0.1.0] - 2026-07-12

### Added

- Tenant, domain, resource-definition and encrypted resource control plane.
- Consumer APIs for tenant identification and scoped resource resolution.
- OIDC admin sessions, CSRF protection, deny-by-default RBAC and break-glass.
- Transactional audit, outbox, signed webhooks and operational reports.
- Governed API clients with scopes, expiry, rotation, usage and global rate limit.
- Embedded multilingual admin console and isolated Playwright catalog.

### Security

- AES-256-GCM keyring and resumable offline rewrap workflow.
- SSRF-resistant webhook delivery, secret-safe caching and defensive HTTP headers.
- Public disclosure policy and repository-wide secret scanning.
- Patched Vitest, Go JOSE, `x/crypto` and `x/net` dependency advisories before
  the first public release.
