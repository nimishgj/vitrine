# Vitrine

Run AI coding agents with read-only access to everything, enforced by the operating system rather than by the agent.

Type `claude`, `codex`, or `pi` as usual. Vitrine runs the real agent inside a native sandbox where:

- your home directory, credentials, and dotfiles do not exist,
- the current project is writable only if you said so, once,
- remote systems are reached only through short-lived read-only identities,
- every session and every grant is recorded in a tamper-evident log.

No containers, no VMs, no overhead. macOS uses a dedicated system user plus Seatbelt; Linux uses bubblewrap plus Landlock.

## Install

```sh
go install github.com/nimishgj/vitrine/cmd/vitrine@latest
vitrine init            # Linux
sudo vitrine init       # macOS, once, creates the sandbox user
export PATH="$HOME/.vitrine/bin:$PATH"
vitrine doctor
```

`vitrine doctor` runs a probe inside the sandbox and shows what was denied and what was allowed. Every row should read `ok`.

## Use

```sh
cd ~/code/myproject
claude                  # first run asks: allow writes here, read-only, or cancel
vitrine grants          # what agents may touch
vitrine audit           # what happened
vitrine run -- ./my-agent --flag
```

## Documentation

- [How it works](docs/how-it-works.md): the shim, the session, what the sandbox contains, and the macOS and Linux backends.
- [Security model](docs/security-model.md): what is guaranteed, what a write grant really means, tamper evidence, and the known limits of this version.
- [Grants and the audit log](docs/grants-and-audit.md): grant matching, the first-run prompt, refused paths, the audit chain, and tamper detection.
- [Command reference](docs/commands.md): every command, file, and configuration key.
- [Platforms](docs/platforms.md): setup, prerequisites, and uninstall for macOS and Linux.

## Status

Early. Local filesystem isolation works on macOS and Linux. Kubernetes and ClickHouse read-only identities are next, followed by Postgres and AWS, an opt-in container backend, and managed enforcement.

## Development

```sh
make test          # unit tests
make integration   # real sandboxes; see docs/platforms.md for prerequisites
make build
```
