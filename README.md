# Vault Snapshot Agent

A custom Vault Agent for managing snapshots automatically.

## Features

- (TBD) Scheduling - Running agent without an external Cron support
- Retention - Keeping only the last N snapshots
- Storage - Destination storages for created snapshots.
    - Local
    - (TBD) GCS

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

Example:

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
