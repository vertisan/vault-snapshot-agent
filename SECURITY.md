# Security Policy

## Supported Versions

Only the latest released version receives security fixes. Please upgrade before
reporting an issue against an older release.

| Version | Supported |
| ------- | --------- |
| Latest release | ✅ |
| Anything older | ❌ |

## Reporting a Vulnerability

**Please do not open a public issue for security problems.**

Report vulnerabilities privately through GitHub Security Advisories:

<https://github.com/vertisan/vault-snapshot-agent/security/advisories/new>

Please include:

- affected version or commit,
- a description of the impact,
- reproduction steps or a proof of concept,
- any suggested mitigation.

You can expect an initial acknowledgement within 7 days and a status update
within 30 days. If a fix is released, you will be credited in the advisory
unless you ask otherwise.

## Handling Credentials

`vault-snapshot-agent` reads Vault AppRole credentials (`roleId`, `secretId`)
from its configuration file and writes Vault snapshots, which contain the full
contents of the Vault store, to the configured destination.

- Keep the configuration file readable only by the account running the agent
  (`chmod 0600`, owned by that account).
- Never commit a configuration file containing real `roleId`/`secretId` values.
- Treat the snapshot destination — local directory or GCS bucket — as being as
  sensitive as Vault itself. Restrict access and enable encryption at rest.
- Scope the AppRole to the minimum policy required to take raft snapshots.

## Supply Chain Security

Every release publishes verifiable supply chain metadata:

- **Keyless signatures** — `checksums.txt` is signed with
  [Sigstore](https://www.sigstore.dev/) `cosign` using a short-lived
  certificate bound to the release workflow's OIDC identity. There is no
  long-lived private key. The signature bundle is published as
  `checksums.txt.sigstore.json`.
- **SBOMs** — an SPDX JSON Software Bill of Materials is generated with
  [syft](https://github.com/anchore/syft) for every release archive and
  published as `<archive>.sbom.json`. Container images carry an SBOM
  attestation on the image index.
- **Build provenance** — SLSA provenance attestations are produced by
  [`actions/attest-build-provenance`](https://github.com/actions/attest-build-provenance)
  for both the release artifacts and the published container image digests.
- **Signed container images** — `ghcr.io/vertisan/vault-snapshot-agent` images
  are signed with keyless `cosign`.

See [Verifying releases](README.md#verifying-releases) in the README for
copy-pasteable verification commands.
