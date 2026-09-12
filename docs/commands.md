# Command reference

All state lives in `~/.vitrine`. Every command that changes state appends to
the audit log.

## Setup

### `vitrine init`

Creates `~/.vitrine`, installs the agent shims in `~/.vitrine/bin`, writes a
default config, and prints the `PATH` line to add. Safe to re-run.

On macOS, run it once as root: `sudo vitrine init`. That creates the hidden
`vitrine` system user, its home at `/var/lib/vitrine`, the session directory,
and a sudoers rule at `/etc/sudoers.d/vitrine` allowing your user to run
`vitrine launch` as the `vitrine` user and nothing else. It then re-runs the
unprivileged half as you. The rule names the absolute path of the binary you
ran, so re-run `sudo vitrine init` if you move or reinstall Vitrine.

### `vitrine doctor`

Runs the isolation probe inside the real sandbox and prints one line per
check. Exit 0 when every check matches its expectation, 2 if something that
must be denied was allowed, 1 if the backend is unavailable. The result is
logged. Run it after install and after any system update.

### `vitrine status`

Shows the backend in use and whether it is available, the enforcement level,
grants (flagging any not recognised by the audit chain), targets, and the last
doctor result.

## Running agents

### `claude`, `codex`, `pi`

With `~/.vitrine/bin` at the front of `PATH`, these run the real agent inside
the sandbox in the current directory. Arguments pass through unchanged. The
agent's login and settings persist in its own state home across sessions.

### `vitrine run [--agent NAME] -- CMD [ARGS...]`

Runs any command inside the sandbox. `--agent` selects a built-in profile
(`claude`, `codex`, `pi`); without it the command gets a generic profile keyed
by its basename, with its own persistent state home. The exit code of the
command is returned.

## Grants

### `vitrine grant PATH [--read-only]`

Allows agents to write (default) or only read under `PATH`. Refuses home,
dot-directories directly under home, system roots and `~/.vitrine`. On macOS
applies the access control entries the `vitrine` user needs.

### `vitrine revoke PATH`

Removes the grant at exactly `PATH` and its access control entries.

### `vitrine grants`

Lists grants with mode, source, date and path. Entries added outside Vitrine
are marked `UNRECOGNISED`.

### `vitrine grants accept`

Re-validates and records every entry in the grants file as recognised. Fails
if any entry hits the refusal list.

## Targets

Targets are remote systems a provider mints read-only credentials for. No
provider ships in the current version, so `add` reports that no kinds are
available; the commands are here for the providers that follow.

### `vitrine target add NAME --kind KIND [--set key=value ...]`
### `vitrine target list`
### `vitrine target remove NAME`

## Audit

### `vitrine audit [--since DURATION] [--json]`

Prints events oldest first. `--since 24h` filters by age. `--json` prints the
raw lines.

### `vitrine audit verify`

Recomputes the hash chain. Exit 3 and the first broken sequence number on
failure.

## Other

### `vitrine version`

Prints the build version.

### Hidden commands

`vitrine probe`, `vitrine launch` (macOS) and `vitrine _inner` (Linux) are
internal. Doctor and the backends call them; you never need to.

## Files

| Path | Purpose | Mode |
|---|---|---|
| `~/.vitrine/config.toml` | backend, targets, profile overrides | 0600 |
| `~/.vitrine/grants.json` | grants | 0600 |
| `~/.vitrine/audit.jsonl` | hash-chained event log | 0600 |
| `~/.vitrine/audit.head` | latest hash, for truncation detection | 0600 |
| `~/.vitrine/bin/` | agent shims | |
| `~/.vitrine/agents/<name>/home` | agent state homes (Linux) | 0700 |
| `/var/lib/vitrine/agents/<name>/home` | agent state homes (macOS, owned by `vitrine`) | 0700 |
| `/var/lib/vitrine/sessions/` | per-session spec, credentials and scratch (macOS) | 1777 |
| `/etc/sudoers.d/vitrine` | launch rule (macOS) | 0440 |

None of the files under `~/.vitrine` are visible inside the sandbox.

## Configuration

`~/.vitrine/config.toml` is written by Vitrine commands and hash-checked at
every launch. Editing it by hand is logged as a tamper event.

```toml
version = 1
backend = "native"      # the only backend in this version

[[targets]]
name = "prod"
kind = "k8s"
[targets.settings]
context = "prod-eu"

[profiles.claude]
binary = "claude-nightly"   # look up a different binary name on PATH

# system_roots = ["/usr", "/bin", ...]   # override the read-only system roots
```
