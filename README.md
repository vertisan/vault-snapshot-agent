# Vault Snapshot Agent

[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/vertisan/vault-snapshot-agent/badge)](https://scorecard.dev/viewer/?uri=github.com/vertisan/vault-snapshot-agent)

A custom Vault Agent for managing snapshots automatically.

## Features

- Retention - Keeping only the last N snapshots
- Storage - Destination storage for created snapshots.
  - Local
  - GCS (Google Cloud Storage)
- Supply chain security - signed releases, SBOMs and SLSA build provenance,
  all publicly verifiable. See [Verifying releases](#verifying-releases).

## Installation

### Binaries

Download an archive for your platform from the
[releases page](https://github.com/vertisan/vault-snapshot-agent/releases), then
verify it before use — see [Verifying releases](#verifying-releases).

### Container image

Multi-arch images (`linux/amd64`, `linux/arm64`, `linux/arm/v7`) are published to
GitHub Container Registry:

```bash
docker run --rm \
  -v "$PWD/vault-snapshot-agent.yaml:/etc/vault.d/vault-snapshot-agent.yaml:ro" \
  ghcr.io/vertisan/vault-snapshot-agent:latest
```

The default configuration path inside the image is
`/etc/vault.d/vault-snapshot-agent.yaml`; override it with `--config` or the
`VAULT_SNAPSHOT_AGENT_CONFIG` environment variable. The image runs as an
unprivileged user (UID `65532`), so any mounted snapshot directory must be
writable by that UID.

## Verifying releases

Every release is signed with [Sigstore](https://www.sigstore.dev/) keyless
signing, ships an SPDX SBOM per artifact, and carries SLSA build provenance.
No shared secret or long-lived key is involved — the signing identity is a
short-lived certificate bound to this repository's release workflow, and the
signature is recorded in the public Rekor transparency log.

You will need [`cosign`](https://github.com/sigstore/cosign) and, optionally,
[`gh`](https://cli.github.com/) and [`grype`](https://github.com/anchore/grype).

Pick the version you want to verify:

```bash
export VERSION="$(gh release list -L 1 -R vertisan/vault-snapshot-agent --json tagName -q '.[].tagName')"
export BASE="https://github.com/vertisan/vault-snapshot-agent/releases/download/$VERSION"
export IDENTITY="https://github.com/vertisan/vault-snapshot-agent/.github/workflows/release.yaml@refs/tags/$VERSION"
```

### 1. Verify the signature on `checksums.txt`

Every other asset is listed in `checksums.txt`, so verifying this one signature
establishes trust in the whole release.

```bash
curl -sSLO "$BASE/checksums.txt"
curl -sSLO "$BASE/checksums.txt.sigstore.json"

cosign verify-blob \
  --certificate-identity "$IDENTITY" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  --bundle checksums.txt.sigstore.json \
  checksums.txt
```

Expected output: `Verified OK`.

### 2. Verify the artifact you downloaded

```bash
curl -sSLO "$BASE/vault-snapshot-agent_${VERSION}_linux_amd64.tar.gz"
sha256sum --ignore-missing -c checksums.txt
```

### 3. Inspect the SBOM

```bash
curl -sSLO "$BASE/vault-snapshot-agent_${VERSION}_linux_amd64.tar.gz.sbom.json"
sha256sum --ignore-missing -c checksums.txt

grype "sbom:vault-snapshot-agent_${VERSION}_linux_amd64.tar.gz.sbom.json"
```

### 4. Verify build provenance

```bash
gh attestation verify --owner vertisan "vault-snapshot-agent_${VERSION}_linux_amd64.tar.gz"
```

This confirms the archive was built by this repository's release workflow, from
a specific commit, on a GitHub-hosted runner.

### 5. Verify the container image

```bash
cosign verify \
  --certificate-identity "$IDENTITY" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  "ghcr.io/vertisan/vault-snapshot-agent:$VERSION"

gh attestation verify --owner vertisan "oci://ghcr.io/vertisan/vault-snapshot-agent:$VERSION"

cosign download sbom "ghcr.io/vertisan/vault-snapshot-agent:$VERSION"
grype "ghcr.io/vertisan/vault-snapshot-agent:$VERSION"
```

> If you would rather not pin the exact tag, replace `--certificate-identity`
> with
> `--certificate-identity-regexp '^https://github\.com/vertisan/vault-snapshot-agent/\.github/workflows/release\.yaml@refs/tags/'`.

## Configuration

### Vault

- `addr` - Vault HTTPS address
- `roleId` - Role ID used to authenticate in Vault API.
- `secretId` - Secret ID used to authenticate in Vault API.
- `approle` - Approle name used to authenticate in Vault API. Defaults to `approle`.

### Storage

- `retention` - The number of snapshots to retain.

#### Local Path

- `path` - A fully qualified path name to the directory where snapshots will be saved, e.g. `/mnt/snapshots`.

#### GCS (Google Cloud Storage)

- `bucket` - The name of the GCS bucket where snapshots will be stored.
- `prefix` - (Optional) A prefix/folder path within the bucket for organizing snapshots, e.g. `vault-snapshots` or `backups/vault`.

## Examples

### Local storage

```yaml
vault:
  addr: "https://127.0.0.1:8200"
  roleId: "05dd3d65-1523-e794-392f-74d387721372"
  secretId: "88936f9e-8ba4-0032-2832-e78788dbc595"
  approle: "approle"
storage:
  retention: 10
  local:
    path: "/mnt/vault-snapshots"
```

### Google Cloud Storage

```yaml
vault:
  addr: "https://127.0.0.1:8200"
  roleId: "05dd3d65-1523-e794-392f-74d387721372"
  secretId: "88936f9e-8ba4-0032-2832-e78788dbc595"
  approle: "approle"
storage:
  retention: 10
  gcs:
    bucket: "my-vault-snapshots"
    prefix: "production"
```

### Mixed

```yaml
vault:
  addr: "https://127.0.0.1:8200"
  roleId: "05dd3d65-1523-e794-392f-74d387721372"
  secretId: "88936f9e-8ba4-0032-2832-e78788dbc595"
  approle: "approle"
storage:
  retention: 10
  local:
    path: "/mnt/vault-snapshots"
  gcs:
    bucket: "my-vault-snapshots"
    prefix: "production"
```

## Development

### Running Tests

```bash
go test ./...
```

### Linting

```bash
golangci-lint run
```
