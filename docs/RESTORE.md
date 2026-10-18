# Restoring Vault from snapshots

This guide describes how to restore a HashiCorp Vault cluster from the snapshots created by `vault-snapshot-agent`.
It covers locating a snapshot, picking the right restore scenario, and the seal-specific steps for Shamir key shares and for auto-unseal with GCP Cloud KMS, AWS KMS, Azure Key Vault, Transit and HSM (PKCS#11).

## Table of contents

- [Restoring Vault from snapshots](#restoring-vault-from-snapshots)
  - [Table of contents](#table-of-contents)
  - [How the snapshots work](#how-the-snapshots-work)
  - [Before you start](#before-you-start)
  - [Step 1: Pause the agent](#step-1-pause-the-agent)
  - [Step 2: Locate and fetch a snapshot](#step-2-locate-and-fetch-a-snapshot)
    - [Local storage](#local-storage)
    - [Google Cloud Storage](#google-cloud-storage)
    - [Check the file](#check-the-file)
  - [Step 3: Choose a restore scenario](#step-3-choose-a-restore-scenario)
    - [Scenario A: rollback on the same cluster](#scenario-a-rollback-on-the-same-cluster)
    - [Scenario B: rebuild / new cluster](#scenario-b-rebuild--new-cluster)
  - [Step 4: Restore according to the seal type](#step-4-restore-according-to-the-seal-type)
    - [Shamir key shares](#shamir-key-shares)
    - [Auto-unseal: common procedure](#auto-unseal-common-procedure)
    - [GCP Cloud KMS](#gcp-cloud-kms)
    - [AWS KMS](#aws-kms)
    - [Azure Key Vault](#azure-key-vault)
    - [Transit (another Vault cluster)](#transit-another-vault-cluster)
    - [HSM / PKCS#11 (Vault Enterprise)](#hsm--pkcs11-vault-enterprise)
  - [Step 5: Verify the restored cluster](#step-5-verify-the-restored-cluster)
  - [Changing the seal type during recovery](#changing-the-seal-type-during-recovery)
  - [Troubleshooting](#troubleshooting)
  - [Recommendations](#recommendations)

## How the snapshots work

- The agent calls `GET /v1/sys/storage/raft/snapshot`, so it only works with Vault clusters using **Integrated Storage (Raft)**.
- The snapshot is taken only on the active (leader) node; agents on standby nodes skip the run.
- Every run produces one file named `vault-snapshot-YYYYMMDDHHMMSS.snap`, where the timestamp uses the **local time zone of the host running the agent**.
- The same file is written to every configured storage:
  - Local storage: `<storage.local.path>/vault-snapshot-YYYYMMDDHHMMSS.snap`, file mode `0400`, owned by the user running the agent (`vault` in the bundled systemd unit).
  - GCS: `gs://<storage.gcs.bucket>/<storage.gcs.prefix>/vault-snapshot-YYYYMMDDHHMMSS.snap`.
- With `storage.retention` set, the agent keeps the newest N snapshots (by modification time) and deletes the rest after each run.

A snapshot is a full copy of the Raft data: secrets, policies, auth methods, tokens, leases, the keyring and the **root key encrypted by the seal of the cluster it came from**.
Its content is encrypted by Vault's barrier, but treat it as sensitive anyway.

> [!IMPORTANT]
> A snapshot can only be unsealed with the seal that protected the original cluster at the time the snapshot was taken: the original Shamir unseal key shares, or the same KMS / Transit / HSM key.
> Recovery keys cannot decrypt the root key.
> If the auto-unseal key is permanently deleted, the data cannot be recovered, even from these snapshots.

## Before you start

Make sure you have:

- A Vault binary of the **same or newer** version than the one that created the snapshot.
- A Vault server configuration that uses Integrated Storage (`storage "raft"`).
- Access to the snapshot (filesystem access to the local path, or read access to the GCS bucket).
- Seal material valid **at the time the snapshot was taken**:
  - Shamir: enough original unseal key holders to reach the original threshold.
  - Auto-unseal: access to the exact same key (KMS key, Transit key or HSM key) plus the original recovery key holders.
- A token with `sudo` capability on the restore endpoint:

```hcl
path "sys/storage/raft/snapshot" {
  capabilities = ["update", "sudo"]
}

path "sys/storage/raft/snapshot-force" {
  capabilities = ["update", "sudo"]
}
```

> [!NOTE]
> If you ran `vault operator rekey` (new unseal or recovery key shares) after the snapshot was taken, the restored cluster goes back to the **old** shares that were valid when the snapshot was created.
> Keep previous key shares until every snapshot that depends on them has been rotated out.

## Step 1: Pause the agent

Stop the agent before you restore.
Otherwise retention may delete the snapshot you need, and a snapshot of a broken or half-restored cluster may replace a good one.

```shell
sudo systemctl stop vault-snapshot-agent.timer
sudo systemctl status vault-snapshot-agent.timer
```

If the agent runs from cron, Kubernetes `CronJob` or another scheduler, suspend it there.

## Step 2: Locate and fetch a snapshot

Because the timestamp is part of the file name, sorting by name gives the newest snapshot last.

### Local storage

```shell
# List the snapshots, newest last
sudo -u vault ls -1 /mnt/vault-snapshots/vault-snapshot-*.snap

# Pick the newest one
SNAPSHOT=$(sudo -u vault ls -1 /mnt/vault-snapshots/vault-snapshot-*.snap | tail -n 1)

# Copy it to the node you restore on (the file is 0400, read it as vault or root)
sudo cp "$SNAPSHOT" /tmp/restore.snap
sudo chown "$USER" /tmp/restore.snap
```

### Google Cloud Storage

```shell
# List the snapshots, newest last
gcloud storage ls gs://my-vault-snapshots/production/ | sort

# Download a specific one
gcloud storage cp gs://my-vault-snapshots/production/vault-snapshot-20261018120000.snap /tmp/restore.snap
```

The identity that downloads needs at least `roles/storage.objectViewer` on the bucket.
If object versioning is enabled on the bucket, `gcloud storage ls --all-versions` also shows snapshots already removed by retention.

### Check the file

```shell
# Prints key counts and space usage; fails on a corrupted file
vault operator raft snapshot inspect /tmp/restore.snap
```

`vault operator raft snapshot inspect` works offline and needs no token.
Before picking a snapshot, compare its timestamp with the time of the incident: choose the newest snapshot taken **before** the data was damaged.

## Step 3: Choose a restore scenario

| Scenario                                 | When                                                                                                 | Command                                                            | Seal material needed after restore                                                                                               |
| ---------------------------------------- | ---------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------- |
| **A. Rollback on the same cluster**      | The cluster is healthy and unsealed, you only need older data back (deleted secrets, broken config). | `vault operator raft snapshot restore <file>`                      | None: the cluster stays unsealed (with auto-unseal) or uses the same Shamir keys.                                                |
| **B. Rebuild / new cluster**             | All nodes or the storage were lost, or you restore into a fresh cluster (DR site, test environment). | `vault operator raft snapshot restore -force <file>`               | Original Shamir shares, or access to the original auto-unseal key.                                                               |
| **C. New cluster with a different seal** | The original seal cannot be used on the target (other cloud, other KMS key, Shamir to KMS).          | Restore as in **B** with the original seal, then migrate the seal. | Original seal first, new seal afterwards. See [Changing the seal type during recovery](#changing-the-seal-type-during-recovery). |

`-force` (or the `sys/storage/raft/snapshot-force` endpoint) is needed whenever the snapshot comes from a different cluster, because the keyring and seal of the snapshot do not match the running cluster.
Without `-force` such a restore is refused, which protects you from accidentally overwriting a cluster with data you cannot unseal.

> [!WARNING]
> The restore command returns as soon as Vault validates and accepts the file, not when the restore is finished.
> Vault then tears down its subsystems, swaps the database and starts them again in the background.
> Wait for `post-unseal setup complete` and `vault is unsealed` in the server log before you resume normal operations.

### Scenario A: rollback on the same cluster

```shell
export VAULT_ADDR="https://vault.example.com:8200"
vault login   # with a token allowed to restore

vault operator raft snapshot restore /tmp/restore.snap
```

The restore is applied to the whole cluster through the leader; requests to a standby node are forwarded.
After the restore, all tokens, leases and secrets are exactly as they were when the snapshot was taken, so tokens created later stop working.
Continue with [Step 5](#step-5-verify-the-restored-cluster).

### Scenario B: rebuild / new cluster

The general flow is the same for every seal type, only the init and unseal steps differ:

1. Start **one** node with an empty Raft data directory and the seal configuration of the original cluster.
2. Initialize it. This creates a **temporary** root token and temporary key shares, used only to run the restore.
3. Unseal it (Shamir) or let it auto-unseal, then log in with the temporary root token.
4. Run `vault operator raft snapshot restore -force <file>`.
5. Unseal with the **original** seal material (Shamir) or let it auto-unseal with the original key.
6. Join the other nodes with empty data directories (`vault operator raft join` or `retry_join` in the config).

The temporary root token and key shares stop working after the restore; they can be discarded.
Use the [section for your seal type](#step-4-restore-according-to-the-seal-type) for the exact commands.

## Step 4: Restore according to the seal type

Check which seal the original cluster used: look at the `seal` stanza in the original server configuration, or at `Seal Type` in the output of `vault status` on a surviving node.
No `seal` stanza means Shamir.

### Shamir key shares

Shamir is the default seal: the root key is split into N key shares and M of them (the threshold) are required to unseal each node after every restart and after a restore.

Configuration of the new node (no `seal` stanza):

```hcl
storage "raft" {
  path    = "/opt/vault/data"
  node_id = "vault-1"
}

listener "tcp" {
  address       = "0.0.0.0:8200"
  tls_cert_file = "/etc/vault.d/tls/vault.crt"
  tls_key_file  = "/etc/vault.d/tls/vault.key"
}

api_addr     = "https://vault-1.example.com:8200"
cluster_addr = "https://vault-1.example.com:8201"
```

Restore:

```shell
export VAULT_ADDR="https://vault-1.example.com:8200"

# 1. Initialize with temporary keys; one share is enough, they are thrown away after the restore
vault operator init -key-shares=1 -key-threshold=1 -format=json > /tmp/temporary-init.json

# 2. Unseal with the temporary key and log in with the temporary root token
vault operator unseal "$(jq -r '.unseal_keys_b64[0]' /tmp/temporary-init.json)"
vault login "$(jq -r '.root_token' /tmp/temporary-init.json)"

# 3. Force the restore
vault operator raft snapshot restore -force /tmp/restore.snap

# 4. The node now needs the ORIGINAL unseal keys.
#    Each key holder runs this until the original threshold is reached:
vault operator unseal
```

After the node is unsealed, join the other nodes:

```shell
# On every additional node, with an empty data directory
vault operator raft join https://vault-1.example.com:8200

# Each joined node must be unsealed with the original key shares as well
vault operator unseal
```

Clean up: `shred -u /tmp/temporary-init.json`.

Pitfalls:

- Unsealing with the temporary key after the restore fails: the root key in the snapshot is protected by the original shares.
- The threshold is the one of the original cluster, not the one used for the temporary init.
- If the shares were rekeyed after the snapshot, you need the shares that were valid when the snapshot was taken.

### Auto-unseal: common procedure

With auto-unseal, the root key is encrypted by an external key (cloud KMS, Transit, HSM) and Vault unseals itself on start.
`vault operator init` returns **recovery keys** instead of unseal keys; they authorize sensitive operations (for example `generate-root`) but cannot unseal Vault.

The new node must use **exactly the same key** as the original cluster: the same `seal` stanza with the same key identifiers and credentials that are allowed to use that key.
A new key, even with the same name in another project, account or region, cannot decrypt the snapshot.

```shell
export VAULT_ADDR="https://vault-1.example.com:8200"

# 1. Start the node with the ORIGINAL seal stanza, then initialize it (temporary recovery keys)
vault operator init -recovery-shares=1 -recovery-threshold=1 -format=json > /tmp/temporary-init.json

# 2. The node auto-unseals; log in with the temporary root token
vault login "$(jq -r '.root_token' /tmp/temporary-init.json)"

# 3. Force the restore
vault operator raft snapshot restore -force /tmp/restore.snap

# 4. The node stays or becomes unsealed using the same external key; check it
vault status
```

Then join the other nodes (`vault operator raft join https://vault-1.example.com:8200`); they auto-unseal with the same seal stanza.
After the restore, the **original** recovery keys apply again and the temporary ones are useless.

The following sections only list what is specific to each provider.

### GCP Cloud KMS

```hcl
seal "gcpckms" {
  project    = "my-project"
  region     = "europe-west1"
  key_ring   = "vault-keyring"
  crypto_key = "vault-unseal"
  # credentials = "/etc/vault.d/gcp-sa.json"   # or GOOGLE_APPLICATION_CREDENTIALS / attached service account
}
```

- `project`, `region` (the key ring location, e.g. `global` or `europe-west1`), `key_ring` and `crypto_key` must match the original cluster.
- The service account of the new nodes needs `cloudkms.cryptoKeyVersions.useToEncrypt`, `cloudkms.cryptoKeyVersions.useToDecrypt` and `cloudkms.cryptoKeys.get` on the key, e.g. through `roles/cloudkms.cryptoKeyEncrypterDecrypter` plus `roles/cloudkms.viewer`.
- Key rotation is fine: older key versions stay available for decryption.
- A **destroyed** key version used to encrypt the root key makes the snapshot unrecoverable; a version scheduled for destruction can still be restored (`gcloud kms keys versions restore`) during the destroy-scheduled period.
- For a DR site in another region, the key ring location does not move with the cluster: the new nodes must still reach the original location (or use a `global` key ring).

Quick access check from a new node:

```shell
gcloud kms keys describe vault-unseal --keyring vault-keyring --location europe-west1 --project my-project
```

### AWS KMS

```hcl
seal "awskms" {
  region     = "eu-central-1"
  kms_key_id = "arn:aws:kms:eu-central-1:111122223333:key/1234abcd-12ab-34cd-56ef-1234567890ab"
  # access_key / secret_key, or prefer an instance profile / IRSA / environment variables
}
```

- `kms_key_id` must point to the same key; an alias works only if it still resolves to that key.
- The IAM principal of the new nodes needs `kms:Encrypt`, `kms:Decrypt` and `kms:DescribeKey` on the key, and the key policy must allow it (check this in a different account: both the key policy and the IAM policy must allow cross-account use).
- For a DR region, use an AWS KMS **multi-Region key**: a replica in the DR region shares the key material, so its ARN can be used there; a regular single-Region key cannot be copied.
- A key in `PendingDeletion` can be rescued with `aws kms cancel-key-deletion`; once deleted, the snapshots protected by it are unrecoverable.
- A disabled key can be re-enabled with `aws kms enable-key`.

Quick access check from a new node:

```shell
aws kms describe-key --key-id "arn:aws:kms:eu-central-1:111122223333:key/1234abcd-12ab-34cd-56ef-1234567890ab"
```

### Azure Key Vault

```hcl
seal "azurekeyvault" {
  tenant_id  = "00000000-0000-0000-0000-000000000000"
  vault_name = "my-vault-unseal-kv"
  key_name   = "vault-unseal"
  # client_id / client_secret, or prefer a managed identity
}
```

- `vault_name` (the Key Vault name) and `key_name` must match the original cluster.
- The identity of the new nodes needs the `get`, `wrapKey` and `unwrapKey` key permissions (access policy) or an RBAC role that includes them, e.g. `Key Vault Crypto Service Encryption User`.
- Key versions created by rotation stay usable for unwrapping, as long as they are not disabled.
- Enable soft-delete and purge protection on the Key Vault: a deleted key or Key Vault can then be recovered (`az keyvault key recover`, `az keyvault recover`) within the retention period; a purged key makes the snapshots unrecoverable.
- If the nodes reach Key Vault through a private endpoint or firewall, allow the new nodes' network before starting them.

Quick access check from a new node:

```shell
az keyvault key show --vault-name my-vault-unseal-kv --name vault-unseal
```

### Transit (another Vault cluster)

```hcl
seal "transit" {
  address    = "https://vault-transit.example.com:8200"
  mount_path = "transit/"
  key_name   = "autounseal"
  # token is better provided through the VAULT_TOKEN environment variable
  tls_ca_cert = "/etc/vault.d/tls/transit-ca.pem"
}
```

- The Transit cluster must be running and unsealed, and the `transit` key with the same `mount_path` and `key_name` must exist; it is a separate cluster that this snapshot does **not** contain, so back it up separately.
- The token used by the seal needs this policy on the Transit cluster (an orphan, periodic token is recommended):

```hcl
path "transit/encrypt/autounseal" {
  capabilities = ["update"]
}

path "transit/decrypt/autounseal" {
  capabilities = ["update"]
}
```

- If the Transit cluster was itself restored from a snapshot, restore it first and check that the key still contains the versions used by the snapshot you restore (`min_decryption_version` must not exclude them).
- The Transit key must not be deleted; keep `deletion_allowed = false` on it.

Quick access check from a new node:

```shell
VAULT_ADDR=https://vault-transit.example.com:8200 vault write transit/encrypt/autounseal plaintext=$(echo -n test | base64)
```

### HSM / PKCS#11 (Vault Enterprise)

```hcl
seal "pkcs11" {
  lib         = "/usr/lib/softhsm/libsofthsm2.so"
  slot        = "2305843009213693953"   # or token_label
  pin         = "env://VAULT_HSM_PIN"
  key_label   = "vault-hsm-key"
  mechanism   = "0x1087"
}
```

- Requires a Vault Enterprise binary with HSM support and a valid license on the new nodes.
- The new nodes need the HSM client library, network access to the HSM and a partition/slot that contains the **same** key (`key_label`, or `key_id`), with the same PIN or credentials.
- HSM keys usually cannot be exported: for a DR site, the key must be replicated or restored with the vendor's tooling (HSM backup, cloning, cluster replication) before the restore.
- `mechanism` and other key-related settings must match the ones used by the original cluster.

## Step 5: Verify the restored cluster

```shell
# Sealed = false, HA Mode = active on the leader
vault status

# All expected nodes are voters and one is the leader
vault operator raft list-peers
vault operator raft autopilot state

# Log in with an original auth method or token and read a known secret
vault login -method=oidc
vault kv get secret/known/path
```

Then:

1. If you need a root token and do not have an original one, create it with the **original** key holders: `vault operator generate-root` with unseal keys (Shamir) or recovery keys (auto-unseal), then revoke it when you are done (`vault token revoke <token>`).
2. Check that audit devices still write to their targets (file paths, sockets) on the new nodes.
3. Update DNS / load balancers if the nodes have new addresses.
4. Start the agent again and confirm a new snapshot is written:

```shell
sudo systemctl start vault-snapshot-agent.timer
sudo systemctl start vault-snapshot-agent.service   # optional immediate run
sudo journalctl -u vault-snapshot-agent.service -n 50
```

The agent's AppRole is part of the snapshot, so its `roleId` / `secretId` keep working, unless the `secret_id` was created after the snapshot or has expired since then; in that case, issue a new one and update the agent configuration.

## Changing the seal type during recovery

A snapshot cannot be restored directly into a cluster that uses a different seal or a different key.
When the original seal is still available but cannot be used long-term (other cloud, new KMS key, moving from Shamir to auto-unseal or back), do it in two steps:

1. Restore the snapshot into a cluster configured with the **original** seal, as in scenario B.
2. Perform a regular seal migration on the restored cluster:
   - Shamir to auto-unseal: add the new `seal` stanza and run `vault operator unseal -migrate` with the original unseal keys; they become the recovery keys.
   - Auto-unseal to Shamir: add `disabled = "true"` to the old `seal` stanza and run `vault operator unseal -migrate` with the recovery keys; they become the unseal keys.
   - Auto-unseal to another auto-unseal: keep the old stanza with `disabled = "true"`, add the new stanza and run `vault operator unseal -migrate` with the recovery keys.
3. Once the migration is complete, remove the old seal stanza from the configuration and take a new snapshot; older snapshots still require the old seal.

Follow the [official seal migration guide](https://developer.hashicorp.com/vault/docs/concepts/seal#seal-migration) for the node-by-node order on multi-node clusters.

## Troubleshooting

| Symptom                                                                        | Likely cause                                                                                                                     | Fix                                                                                            |
| ------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| Restore is refused because the keyring or seal does not match                  | The snapshot comes from another cluster                                                                                          | Use `restore -force` (scenario B).                                                             |
| `permission denied` on restore                                                 | The token lacks `sudo` on `sys/storage/raft/snapshot` / `snapshot-force`                                                         | Use the temporary root token or add the policy shown in [Before you start](#before-you-start). |
| Node is sealed after a Shamir restore and the temporary key does not unseal it | Expected: the snapshot uses the original key shares                                                                              | Unseal with the original shares, up to the original threshold.                                 |
| Node does not unseal after an auto-unseal restore                              | The new node cannot use the original key: wrong key ID, missing IAM / RBAC permissions, network blocked, key disabled or deleted | Check the seal errors in the server log and the access check command for your provider.        |
| `vault status` hangs or reports the old state right after the restore          | The restore is still running in the background                                                                                   | Wait for `post-unseal setup complete` in the log.                                              |
| Joined nodes do not appear in `list-peers` or stay sealed                      | Nodes still have old Raft data, or (Shamir) they were not unsealed                                                               | Wipe their data directory, join again and unseal with the original keys.                       |
| Agent fails with `cannot login into Vault with AppRole`                        | `secret_id` was issued after the snapshot or expired                                                                             | Issue a new `secret_id` and update the agent configuration.                                    |
| `snapshot inspect` fails                                                       | The file is truncated or corrupted (e.g. an interrupted copy)                                                                    | Pick an older snapshot or the copy from the other storage.                                     |

## Recommendations

- Test the full restore regularly, e.g. into a throwaway single-node cluster, and time it.
- Store snapshots in at least two places (for example local and GCS), and keep the GCS bucket in another project or region than Vault.
- Protect auto-unseal keys against deletion: GCP IAM conditions and the destroy-scheduled duration, AWS key policies and SCPs, Azure purge protection, `deletion_allowed = false` on Transit keys, HSM backups.
- Keep unseal or recovery key shares, and documentation of the seal configuration, outside the Vault cluster they protect.
- After a rekey or a seal migration, keep the old key material until all older snapshots are removed by retention.
