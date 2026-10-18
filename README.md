# Vault Snapshot Agent

A custom Vault Agent for managing snapshots automatically.

## Features

- Retention - Keeping only the last N snapshots
- Storage - Destination storage for created snapshots.
  - Local
  - GCS (Google Cloud Storage)

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

## Restoring from a snapshot

Snapshots are regular Vault Integrated Storage (Raft) snapshots, restored with `vault operator raft snapshot restore`.
See [docs/RESTORE.md](docs/RESTORE.md) for the full procedure, including restore scenarios and seal-specific steps for Shamir key shares, GCP Cloud KMS, AWS KMS, Azure Key Vault, Transit and HSM (PKCS#11).

## Development

### Running Tests

```bash
go test ./...
```

### Linting

```bash
golangci-lint run
```
