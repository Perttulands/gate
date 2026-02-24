# gate city — PRD

*Mode: prescriptive. This describes what gate city is and must do.*

---

## What It Is

`gate city` is the city-readiness checker. It answers one question: **is this system safe to install inside the city?**

A system that passes `gate city` can be `git clone`d into the city, populated with Polis-owned config, and upgraded later via `git pull` — without risk of data loss, config corruption, or Polis data leaking into the public repo.

It is one half of the gate binary. `gate check` guards main branch quality. `gate city` guards the city wall.

The Centaur embodies both.

`gate city` exists to preserve Polis coherence through structure. It should make unsafe installs hard by default and leave an explicit, inspectable learning trail when a system fails or is partially validated.

---

## The Four Checks

### 1. Boundary Declared
**Question:** Does the repo's `.gitignore` explicitly list every file Polis will own?

A Polis-owned file that is not in `.gitignore` will be committed on the next `git add .` and pushed to the public repo. This is the primary privacy risk.

**How it works:**
- The system declares its Polis-owned files in `city.toml` under `[city].polis_files`
- `gate city` validates each declared path is safe: non-empty, relative, no `..` traversal, no absolute paths
- Coverage check uses Git's own ignore semantics (same behavior as `git add` / `git check-ignore`, including negations and precedence), not ad hoc glob matching
- Pass: all `polis_files` entries are git-ignored
- Fail: any `polis_files` entry is unsafe or not ignored by Git semantics

### 2. Standalone Functional
**Question:** Does `git clone` alone produce a working system?

If a system requires Polis-specific setup to build or run at all, it was never a generic tool — it was always a Polis tool wearing a generic costume. A city-ready system must be complete without Polis context.

**How it works:**
- `city.toml` declares a `standalone_check` command (e.g. `go build ./...`, `./bin --help`, `make check`)
- `gate city` clones the repo to a temp directory (no Polis files present)
- Runs `standalone_check` in the clean clone with isolated env (no Polis secrets injected)
- Enforces timeout (default `120s`, configurable with `--standalone-timeout`)
- Pass: command exits 0
- Fail: command exits non-zero, times out, or clone fails

This check can be skipped with `--skip-standalone` (e.g. for systems with external build dependencies). Skipped checks produce a warning, not a pass.

### 3. Config Hooks Declared
**Question:** Does every Polis customization have an explicit hook?

A tracked file that must be manually edited to work in Polis is a design flaw. It means `git pull` will overwrite Polis config. Every point where Polis behaviour differs from the generic default must be a named, declared hook with a defined fallback.

**How it works:**
- `city.toml` declares config hooks under `[[hook]]` — each names a Polis-owned file and specifies what happens when absent (`fallback = "defaults"` | `"fail"` | `"env:VAR"`)
- `gate city` verifies:
  - Every hook file is listed in `polis_files` (and therefore git-ignored)
  - `env:VAR` fallback names a valid env var token (`[A-Z_][A-Z0-9_]*`)
  - No hook has `fallback = "fail"` unless the file already exists at `--install-at` path (would be a runtime failure on fresh install)
- Pass: all hooks are sound
- Fail: hook file not in `polis_files`, or `fallback = "fail"` with no file present

### 4. Split Real on Disk
**Question:** Do the Polis-owned files exist in place before `git pull` runs?

Even with a perfect `.gitignore`, running `git pull` before Polis files are present can cause subtle failures. This check confirms the install is complete and the boundary is real.

**How it works:**
- Requires `--install-at <path>` pointing to the city install location
- `gate city` walks every entry in `polis_files` and checks for its presence at `<install-at>/<polis_file>`
- Presence semantics:
  - File path: path exists and is a regular file
  - Directory path (trailing `/`): path exists and is a directory
  - Glob pattern: at least one matching path exists
  - Symlink does not satisfy presence by default
- Pass: all declared Polis files/dirs are present by type
- Fail: any declared file is absent or wrong type
- Skipped (with warning) if `--install-at` is not provided

---

## city.toml — The City Contract

Every city-ready system ships a `city.toml` in its repo root. This file belongs to the generic system (it is tracked upstream). It declares the Polis contract without containing any Polis data.

```toml
[city]
schema_version = 1

# Files and directories Polis will own in the install location.
# All must be listed in .gitignore.
polis_files = [
  "polis.yaml",
  "memory/",
  "transcripts/",
  "rulings/",
  ".secrets",
]

# Command run in a clean git clone to verify standalone functionality.
# Must exit 0. Leave empty to skip (produces a warning).
standalone_check = "go build ./..."

# Config hooks: points where Polis behaviour differs from generic defaults.
[[hook]]
file = "polis.yaml"
fallback = "defaults"   # system runs with built-in defaults if absent

[[hook]]
file = ".secrets"
fallback = "env:POLIS_API_KEY"  # falls back to env var if file absent
```

### Rules for city.toml
- It is a tracked file. It belongs upstream. It contains no Polis data.
- `schema_version` is required. Current version is `1`.
- `polis_files` uses paths relative to repo root. Glob patterns allowed (`memory/**`).
- `polis_files` entries must not be absolute, empty, or contain path traversal (`..`).
- `standalone_check` runs in a temp directory with no Polis files. It must not require network access, secrets, or a running database.
- `fallback = "fail"` is permitted but means the system cannot be installed without that file present. Gate will flag this if the file does not exist at `--install-at`.

---

## CLI

```
gate city <repo-path>                      # check city-readiness in place
gate city <repo-path> --install-at <path>  # also run check 4 (split on disk)
gate city <repo-path> --skip-standalone    # skip check 2 (produces warning)
gate city <repo-path> --standalone-timeout 120s
gate city <repo-path> --json               # machine-readable verdict
```

## The Verdict

```json
{
  "pass": false,
  "status": "fail",
  "repo": "relay",
  "checks": [
    {"name": "boundary",    "status": "pass", "detail": "12 polis_files all ignored by Git semantics"},
    {"name": "standalone",  "status": "skip", "detail": "skipped by --skip-standalone"},
    {"name": "config-hooks","status": "fail", "detail": ".secrets has fallback=fail but file absent at install path"},
    {"name": "split",       "status": "fail", "detail": "memory/ missing at /home/polis/tools/relay/memory/"}
  ],
  "summary": {"pass": 1, "fail": 2, "skip": 1},
  "exit_code": 1
}
```

Exit code precedence:
- `1` = one or more failed checks
- `2` = no failed checks, but one or more skipped checks
- `0` = all checks pass, none skipped
- `3` = invalid contract/input (e.g. malformed `city.toml`)

---

## Bead Recording

Every `gate city` run records a bead, same pattern as `gate check`. It captures enough detail for replay and learning:
```
br create "gate city: <repo> (<status>)" -t gate \
  --labels "tool:gate,kind:city,status:<pass|warn|fail>,repo:<name>" \
  --description "<per-check verdicts, skips, durations, and concrete remediations>"
```

`gate history` returns both check and city verdicts — the full gate record for a repo.

---

## Failure Paths

| Check | Fail means | Fix |
|---|---|---|
| boundary | Polis data could leak to public repo | Add missing paths to `.gitignore` upstream |
| standalone | System is Polis-specific, not generic | Remove Polis dependencies from core system |
| config-hooks | `git pull` will overwrite Polis config | Add config hook + fallback to system code |
| split | Install is incomplete | Create the missing Polis-owned files |

---

## What This Does NOT Do

- It does not install the system. That is a human action.
- It does not create Polis-owned files. It only checks their presence.
- It does not modify `.gitignore`. It only reads it.
- It does not decide whether a system should enter the city. It reports. The caller decides.

---

## Success Criteria

- `gate city` on a compliant repo completes in under 10 seconds (standalone check excluded)
- `gate city` on a non-compliant repo produces clear, actionable failure messages
- A system that passes `gate city` can safely receive `git pull` without Polis data risk
- `city.toml` can be written and understood by any engineer in under 5 minutes
- Gate checks itself — `gate` passes its own `gate city` before being installed in the city
- `gate city` bead records allow another citizen to understand exactly why a run passed, warned, or failed without rerunning it
