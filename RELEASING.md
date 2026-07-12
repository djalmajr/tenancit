# Releasing Tenancit

This checklist keeps public releases reproducible and free of private material.
The first public version is `v0.1.0`; creating a tag and changing repository
visibility require explicit maintainer authorization.

## Candidate

- [ ] `main` is clean and matches `origin/main`.
- [ ] Three consecutive CI runs pass without manual reruns.
- [ ] Security, CodeQL (when public), dependency and secret scans pass.
- [ ] `LICENSE`, `SECURITY.md`, `CONTRIBUTING.md`, README and changelog agree on
      O'Saasy and the supported-version policy.
- [ ] `gitleaks git --redact --config .gitleaks.toml` and `git fsck --full` pass.
- [ ] Documentation builds and public links contain no local/private references.
- [ ] Container image is built from the candidate SHA and recorded by digest.

## Publish

- [ ] Obtain explicit authorization to make the repository public.
- [ ] Confirm branch rules, Dependabot, CodeQL and private vulnerability
      reporting are active after the visibility change.
- [ ] Tag the verified SHA as `v0.1.0` and create release notes from
      `CHANGELOG.md`; never rebuild a different SHA for the release.
- [ ] Verify a clean clone can run `make test`, `make build` and the documented
      quick start without maintainer-local files.

## After release

- [ ] Confirm the security policy and issue forms are reachable anonymously.
- [ ] Confirm the GitHub release records its URL, tag, SHA, image digest and CI
      evidence; durable follow-ups belong in the roadmap, not a session handoff.
- [ ] Move released changelog entries out of `Unreleased` before tagging.
