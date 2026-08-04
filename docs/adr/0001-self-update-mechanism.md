# ADR 0001: Self-update mechanism

- **Status:** Proposed
- **Date:** 2026-08-04
- **Deciders:** maintainers

## Context

`vault-snapshot-agent` is a one-shot CLI. It is installed on every node of a Vault cluster and
fired hourly by a systemd timer ([`docs/systemd/vault-snapshot-agent.timer`](../systemd/vault-snapshot-agent.timer)).
Each run asks Vault whether the local node is the leader (`internal/vault/vault.go`), and only the
leader takes and ships a snapshot. That design is what makes it safe to install the same unit
cluster-wide — and it is also why an upgrade is an N-node operation, not a one-node operation.

**There is currently no upgrade path.** The only distribution channel is a tarball attached to a
GitHub Release:

- no deb/rpm — `.goreleaser.yaml` has no `nfpms:` block
- no container image — `Dockerfile` is a one-line `FROM alpine` stub, and `.goreleaser.yaml` has no
  `dockers:` block, despite `.github/workflows/release.yaml:29-34` logging in to `ghcr.io`
- no Homebrew tap, no Helm chart, no Ansible role

Meanwhile the release cadence is roughly weekly. Renovate auto-merges dependency bumps, which
semantic-release turns into a patch release, which triggers GoReleaser — versions 1.0.6 through
1.0.11 landed in five weeks, every one of them a `fix(deps)` commit.

So operators are asked to hand-swap a tarball on every Vault node, about weekly, forever. Nobody
does that. And they cannot even tell they are behind: `--version` prints a number
(`main.go:13-17`) with nothing to compare it against.

This ADR asks whether the answer is a **self-update command** — `vault-snapshot-agent update`,
which downloads a newer release and replaces the running binary in place.

## Facts this decision rests on

Verified against the repository and the published `1.0.11` release, since several are easy to
assume wrongly.

| Fact | Source |
|---|---|
| Assets are named `vault-snapshot-agent_{version}_{os}_{arch}.tar.gz`, arch ∈ `386`, `amd64`, `arm64`, `armv6`, `armv7` | release `1.0.11` assets |
| Archives are `.tar.gz` on **every** OS, including Windows — not `.zip` | release `1.0.11` assets |
| `checksums.txt` is plain `sha256sum` format (`<hex>` + two spaces + filename) | release `1.0.11` |
| Tags are **bare — no `v` prefix** (`1.0.11`, not `v1.0.11`) | `.releaserc.json` → `"tagFormat": "${version}"` |
| The `develop` branch publishes **prereleases** | `.releaserc.json` → `branches` |
| `.goreleaser.yaml` has **no `archives:` block** — asset naming comes from GoReleaser's *default* template, i.e. it is an unpinned contract that can shift under a GoReleaser upgrade | `.goreleaser.yaml` |
| Releases are **unsigned** — no `signs:`, no SBOM, no build attestation | `.goreleaser.yaml`, `.github/workflows/release.yaml` |
| GoReleaser's default ldflags set `main.version`, `main.commit`, `main.date`, `main.builtBy`. Only `version` exists (`main.go:13`); the other three are silently dropped by the linker | `main.go` |
| The repo has **no `net/http` usage, no semver library, no checksum or signature verification, and no atomic-write helper** anywhere | whole repo |
| Library code calls `log.Fatal` / `os.Exit` instead of returning errors | `internal/storage/storage_driver_local.go:24`, `internal/vault/vault.go:63`, `pkg/agent/agent.go` |
| Config is strict-parsed YAML with no proxy, CA-bundle, or network settings | `internal/config/config.go`, `internal/utils/yaml_parser.go` |

### The systemd constraint, stated precisely

It is tempting to say "`ProtectSystem=full` makes `/usr` read-only, so self-update is impossible."
That is half right, and the distinction matters:

`ProtectSystem=` applies **only inside the service's own mount namespace**. So:

- **Operator-invoked** — `sudo vault-snapshot-agent update` from a shell is *not* subject to the
  unit's sandbox. It can write `/usr/bin/vault-snapshot-agent` today, with no unit changes at all.
- **Service/timer-invoked** — an auto-update triggered from inside the unit *is* blocked, by
  `ProtectSystem=full` plus `User=vault` plus `NoNewPrivileges=yes`
  ([`docs/systemd/vault-snapshot-agent.service`](../systemd/vault-snapshot-agent.service)). It
  should stay blocked.

One further correction: `/usr/local` is **under `/usr`**, so relocating the binary to
`/usr/local/bin` does *not* by itself escape `ProtectSystem=full`. Relocation only buys something
if the goal is *unprivileged* (non-root) updates — see [Install path](#install-path).

## Drivers

### For a self-update mechanism

1. **There is no upgrade path at all.** Any mechanism beats the status quo.
2. **Staying current is the security story here.** The release churn *is* dependency-CVE hygiene —
   almost every release is a `fix(deps)` bump. A fleet drifting on old builds is precisely the risk
   worth addressing for a tool that holds Vault AppRole credentials.
3. **Operators cannot discover drift.** There is no signal that a newer version exists.
4. **Fleet arithmetic.** N Vault nodes × weekly releases is real, recurring toil.

### Against

1. **Self-update is a distribution mechanism of last resort.** It re-implements — worse — what apt
   and yum already do: signature verification against a trusted repo key, dependency handling,
   rollback, inventory, and an audit trail. On Linux servers running Vault, configuration
   management is near-certainly already in place.
2. **It expands attack surface exactly where it hurts most.** This binary holds Vault AppRole
   credentials and writes the backups of last resort. Giving it the ability to fetch and execute
   new code from the internet converts a compromise of the GitHub account or the release workflow
   into automatic, fleet-wide remote code execution on the Vault cluster.
3. **`checksums.txt` does not provide the property people assume it does.** It is served from the
   same origin as the artifact it describes. An attacker who can replace the tarball can replace
   the checksum file in the same motion. It protects against truncated and corrupted downloads —
   *not* against a compromised release. The same applies to the `digest` field GitHub returns on
   release assets. Shipping checksum verification and describing the result as "verified" is worse
   than making no claim, because it buys operator trust the mechanism has not earned.
4. **Network egress.** Vault servers routinely sit in segments with no route to `github.com`. Today
   the agent talks only to Vault (usually local) and GCS. Adding a GitHub dependency is a new
   firewall and proxy requirement, and `internal/config/config.go` has no surface for proxies or
   custom CA bundles.
5. **Failure is silent.** A half-applied update breaks the *next hourly snapshot*, and the
   `log.Fatal` habit throughout the codebase means it dies without a health signal. Snapshots stop
   and nobody finds out until a restore is needed.
6. **The marginal value is small once packages exist.** If you have `sudo` on the box to run
   `update`, you have `sudo` to run `apt upgrade`.

## Options considered

### Option A — OS packages (deb/rpm) as the primary channel

Add an `nfpms:` block to `.goreleaser.yaml`. GoReleaser builds deb and rpm from the same binaries
already being produced, and can install the systemd units that currently sit unmanaged in
`docs/systemd/`.

- **+** Upgrades become `apt upgrade` / `yum update` — no new code, no new attack surface.
- **+** Package signatures verify against a repo key the operator already trusts.
- **+** Config management, inventory, and rollback all work unchanged.
- **+** The systemd units become a managed, versioned artifact instead of copy-paste docs.
- **−** Requires an apt/yum repository to get *automatic* updates. Without one, operators still
  fetch the `.deb` manually — better than a tarball, but not self-serving.

### Option B — Container image

Make the `Dockerfile` real and add `dockers:` to `.goreleaser.yaml`. The ghcr.io login in
`release.yaml:29-34` already exists and currently publishes nothing.

- **+** ~10 lines of config. Upgrades are a tag bump, handled by whatever already runs containers.
- **−** Does not fit the deployment model. The agent is a systemd-timer job on the Vault hosts
  themselves, and leader detection assumes it runs beside Vault.

### Option C — Notify-only version check

A `version --check` subcommand that queries the GitHub Releases API, compares semver, and reports
drift. Read-only; never touches the binary.

- **+** Closes the discovery gap, which is half the actual problem.
- **+** No privilege, no writes, no meaningful new attack surface beyond an outbound GET.
- **+** Small, testable, and reversible.
- **−** The operator still has to perform the upgrade.

### Option D — Full self-update (`update` subcommand)

Download the matching asset, verify it, and atomically replace the running binary.

- **+** Genuinely one-command upgrades, no external infrastructure.
- **−** Every objection in [Against](#against) — most sharply, an unsigned release pipeline means
  the verification step would be theatre.

## Decision

**Do not make binary-replacing self-update the answer to the distribution gap.** Fix distribution
first, add notification second, and treat full self-update as an opt-in third stage that is gated
on release signing.

Concretely, in order:

### Stage 0 — prerequisites (worth doing regardless of the self-update decision)

| Change | File | Why |
|---|---|---|
| Add `commit` and `date` vars so GoReleaser's *existing* default ldflags actually land | `main.go` | Today they are injected into symbols that do not exist; `--version` is undiagnosable |
| Add an explicit `archives:` block | `.goreleaser.yaml` | Pins the asset-name contract. Any updater — or any documented `curl` command — depends on it, and today it is a GoReleaser default that can shift under an upgrade |
| Add release signing: cosign keyless `signs:`, or `actions/attest-build-provenance` | `.goreleaser.yaml`, `.github/workflows/release.yaml` | The gate for any automated update. Without it there is no authenticity, only corruption detection |
| Add `nfpms:` for deb + rpm (Option A) | `.goreleaser.yaml` | The real fix for "how do I upgrade". Also brings `docs/systemd/` under version management |
| Make the `Dockerfile` real, or add `dockers:` (Option B) | `Dockerfile`, `.goreleaser.yaml` | The ghcr.io login already runs on every release and publishes nothing |

Stage 0 alone closes most of the gap this ADR was raised to address.

### Stage 1 — `version --check` (Option C)

Cheap, safe, and it solves the discoverability half of the problem.

**It must be opt-in only, never automatic on every run.** The unauthenticated GitHub API limit is
60 requests/hour per source IP. An hourly timer across a cluster behind one NAT address will start
receiving `403`s, and a snapshot agent must never fail because a version check did.

### Stage 2 — `update` (Option D), only if Stage 0 signing has landed and operators still want it

Opt-in, operator-invoked, **never** timer-invoked. Design sketched below so this is a known
quantity rather than a blank cheque.

## Design sketch, if Stage 2 proceeds

### CLI surface

Register subcommands on the existing `urfave/cli` v3 root command in `main.go`, keeping the root
`Action` so a bare `vault-snapshot-agent` still takes a snapshot (backwards compatibility).

```
vault-snapshot-agent version [--check]
vault-snapshot-agent update  [--check] [--dry-run] [--version=X] [--force] [--yes] [--allow-prerelease]
```

### Release discovery

`GET https://api.github.com/repos/vertisan/vault-snapshot-agent/releases/latest` — note this
endpoint **excludes prereleases automatically**, which handles the `develop` branch for free.
`/releases/tags/{tag}` for `--version=X`.

Decode the handful of needed fields with `encoding/json`. **Do not add `google/go-github`** — it is
one endpoint, and dependency minimalism matters especially for a project whose entire release
cadence is dependency churn. Honour `GITHUB_TOKEN` for the 5000/hr authenticated limit; never
require it.

### Version comparison

Use `golang.org/x/mod/semver` — stdlib-adjacent, already ubiquitous as an indirect dependency, and
avoids taking on `Masterminds/semver` or `blang/semver` as a direct one. Its API requires the `v`
prefix and our tags lack it, so a one-line `"v" + tag` adapter is needed.

`version == "dev"` (the default in `main.go:13`, and the value `main_test.go` asserts) must **refuse
to update** with a clear message unless `--force` — a local build has no meaningful ordering against
a release.

### Asset selection

Exact-name match for `386`, `amd64`, and `arm64`. For `GOARCH=arm`, `armv6` vs `armv7` is **not
knowable from `runtime` alone** — read the `GOARM` build setting from `debug.ReadBuildInfo()`.

Recommend **not supporting Windows self-update** at all, and erroring out cleanly: a running `.exe`
cannot be overwritten, the rename-self workaround is fiddly and hard to test, and the real
deployment target is Linux under systemd.

### Verification

Stream the download through `sha256` via `io.TeeReader` and match the corresponding
`checksums.txt` line. Document in the same breath — in the command's own output, not just here —
that this is **corruption protection, not authenticity**, until Stage 0 signing lands. Once it
does, verify the signature and treat the checksum as a secondary check.

Use an `http.Client` with an explicit `Timeout` and the default transport (so `HTTPS_PROXY` is
honoured). Do not pin certificates; GitHub rotates them.

### Download and replace

- Resolve the real target with `os.Executable()` followed by `filepath.EvalSymlinks()`.
- **Preflight the containing directory for write permission** and fail early, naming the path and
  the effective uid. This is the single most likely failure mode and deserves a good error rather
  than a stack trace at the last step.
- Extract only the single binary entry from the tar.gz, with `..`-traversal and
  decompressed-size guards.
- Write to a temp file **in the same directory** as the target, so the filesystem is guaranteed to
  match and `os.Rename` is atomic. Preserve the existing file's mode.
- **Smoke-gate before the rename:** exec the new binary's `--version` and confirm it reports the
  expected version. Cheap, and it catches a truncated or wrong-arch download before it becomes the
  live binary.
- Keep the previous binary as `<name>.old` for rollback.
- Take an advisory `flock` so an update cannot race the hourly timer-fired snapshot.
- Replacing a running binary via `rename` is safe on Linux — the running process keeps the old
  inode and finishes normally.

### Error handling

The new package must **return errors**. Do not follow the existing `log.Fatal`-inside-library
pattern (`internal/storage/storage_driver_local.go:24`, `internal/vault/vault.go:63`) — an update
path needs to unwind and roll back, which `os.Exit` makes impossible.

### Install path

If unprivileged (non-root) updates are a goal, `/usr/bin` and `/usr/local/bin` are both wrong —
both sit under `/usr`, and neither is writable by the `vault` user. The target would be a dedicated
directory such as `/opt/vault-snapshot-agent/bin`, owned by an operator group, with
`ReadWritePaths=` scoped to it *only if* the service itself ever needs to write there.

If updates stay operator-invoked under `sudo` — the recommendation — **no relocation is needed**,
because the unit's sandbox does not apply to a shell invocation. In that case leave
`docs/systemd/vault-snapshot-agent.service` hardened exactly as it is.

### Testing

- `httptest` server returning canned release JSON, a generated tar.gz, and a matching
  `checksums.txt`. This would be the repo's first `httptest` use and first `testdata/` directory.
- Table-driven `goos`/`goarch`/`GOARM` → expected filename, in the style of
  `internal/utils/yaml_parser_test.go`.
- Table-driven semver comparison covering `dev`, prereleases, equal versions, and downgrades.
- Checksum-mismatch test asserting the target file is **byte-identical afterwards** and no `.old`
  is left behind.
- `.golangci.yml` excludes `*_test.go`, but `errcheck` applies to the new non-test code — every
  `Close()` must be handled.

### File layout

```
internal/update/release.go     + release_test.go     GitHub Releases API client
internal/update/checksum.go    + checksum_test.go    checksums.txt parsing and verification
internal/update/archive.go     + archive_test.go     tar.gz extraction with guards
internal/update/replace.go     + replace_test.go     atomic replace, smoke gate, rollback
internal/update/testdata/
```

Modified: `main.go`, `.goreleaser.yaml`, `README.md`, and `docs/systemd/*.service` only if the
install path changes.

## Consequences

**If Stage 2 proceeds, the project takes on:**

- Signing infrastructure, plus a key or keyless-identity policy that has to be documented and kept
  working — a permanent operational commitment, not a one-off config change.
- A network egress requirement (`github.com`) on Vault nodes that currently need none, and the
  proxy/CA configuration surface to go with it.
- Rollback support burden, including the `.old` binary lifecycle and disk usage.
- A supported install-path contract, if unprivileged updates are in scope.
- The residual risk that a release-pipeline compromise reaches every Vault node automatically.

**If it does not proceed:**

- Packaging work happens in Stage 0 instead, which is less code and less risk for more of the
  benefit.
- Operators continue upgrading through configuration management — which, for this class of tool on
  this class of host, is where the upgrade belongs.

**Either way, Stage 0 and Stage 1 should happen.** They are the parts that are unambiguously worth
doing, and Stage 0 is a prerequisite for Stage 2 ever being safe.

## Follow-up items surfaced while writing this

Small, unrelated to the decision, worth tracking separately:

- `main.go` is missing `commit` / `date` / `builtBy` vars, so three of GoReleaser's four default
  ldflags are silently discarded.
- `.github/workflows/release.yaml:29-34` logs in to `ghcr.io` on every release but nothing pushes
  an image.
- `Dockerfile` is a `FROM alpine` stub that builds nothing and is referenced by no workflow.
- `internal/config/config.go:16,19` carries `default:"..."` struct tags, but nothing reads them —
  defaulting happens imperatively in `internal/vault/vault.go`.
