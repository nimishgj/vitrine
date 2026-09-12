# How Vitrine works

Vitrine makes a coding agent read-only by putting it behind boundaries the
operating system enforces. The agent's own permission prompts, allowlists and
good intentions play no part. This page walks through what happens from the
moment you type `claude` to the moment the session ends.

## 1. The shim

`vitrine init` installs symlinks named `claude`, `codex` and `pi` in
`~/.vitrine/bin`, all pointing at the `vitrine` binary. With that directory at
the front of your `PATH`, typing `claude` runs Vitrine. It looks at its own
`argv[0]`, recognises the agent name, and searches the rest of your `PATH`
(skipping its own shim directory) for the real binary. Symlinks are resolved
so the sandbox executes the real file.

Any other program runs the same way through `vitrine run -- CMD ARGS`. An
agent profile is just a name, a binary to look up and the dot-directories the
agent keeps its state in; a command without a profile gets a generic one keyed
by its basename.

## 2. The session

Every launch is a session with these steps, in this order. Any failure aborts
before the agent starts; there is no partial session and no way to run the
agent outside the sandbox.

1. **Integrity check.** The grants file and config file are hashed and
   compared with the hashes recorded in the audit chain. A mismatch is logged
   as a tamper event. Grants that appeared outside Vitrine are not honoured
   (see [grants and audit](grants-and-audit.md)).
2. **Profile and binary.** The real agent binary is located and its install
   tree identified, so the sandbox can read it. For a Node package that is the
   outermost `node_modules` directory; otherwise the binary's own directory.
3. **Grant for the current directory.** The nearest ancestor grant is used.
   If none exists and stdin is a terminal, Vitrine asks once: allow writes,
   read-only, or cancel. The answer is stored as a grant and logged. Without a
   terminal the session fails closed.
4. **Credentials.** For each configured remote target, the provider mints a
   short-lived read-only credential as you, on the host. Only the resulting
   files and environment variables enter the sandbox. No provider ships in the
   current version; the interface is in place.
5. **Sandbox spec.** A platform-neutral description: read-only paths,
   read-write paths, denied paths, the complete environment, working
   directory, agent state home and session scratch.
6. **Audit start.** One line is appended to the audit log. If the log cannot
   be written the session does not start.
7. **Run.** The backend isolates the process and blocks until it exits. Your
   terminal is inherited directly, so interactive interfaces work.
8. **Cleanup and audit end.** Credentials are cleaned up, the session
   directory removed, and the exit code logged.

## 3. What the sandbox contains

Inside the sandbox the agent sees:

| Path | Access |
|---|---|
| System and toolchain roots (`/usr`, `/bin`, `/System`, `/Library`, `/opt/homebrew`, `/nix`, … whichever exist) | read |
| The agent's own install tree | read |
| The current directory, if the grant is read-only | read |
| The current directory, if the grant allows writes | read and write |
| `.git/hooks` and `.git/config` inside a write grant | read only |
| The agent's state home (`$HOME` inside the sandbox) | read and write |
| Session scratch (`$TMPDIR` inside the sandbox) and shared temp | read and write |
| Credential files minted for this session | read |
| Everything else, including your real home directory | absent or denied |

The environment is built from scratch. Only `PATH` (minus the shim
directory), `TERM`, `LANG`, `LC_*`, `COLORTERM`, your git name and email, and
credential variables are set. `HOME` points at the agent's state home so its
login and settings persist across sessions without touching your dotfiles.
`VITRINE=1` is set so tools can detect they are sandboxed.

## 4. The backends

The spec is the same everywhere; only the mechanism differs. See
[platforms](platforms.md) for setup.

### macOS: separate user plus Seatbelt

Two layers. The agent runs as a hidden system user named `vitrine`, so your
files are unreadable to it unless a grant added an access control entry. On
top of that, the process runs under a Seatbelt profile generated per session
(`sandbox-exec`, the same mechanism Chrome and App Store apps use). The profile
denies everything by default and then allows exactly the paths in the spec,
resolved to real paths because `/var` and `/tmp` are symlinks on macOS.

Launching as another user goes through a single sudoers rule that
`sudo vitrine init` installs, restricted to `vitrine launch`. The session
directory holding the spec, credentials and scratch is created by you and
shared with the `vitrine` user through an access control entry; no chown and
no further privilege is involved.

### Linux: bubblewrap plus Landlock

One kernel mechanism. bubblewrap builds a fresh mount namespace: read-only
binds of the system roots, a private `/proc`, `/dev` and tmpfs `/tmp`, an empty
tmpfs at your home path so nothing under it exists, and binds only for the
paths in the spec. `.git/hooks` and `.git/config` are re-bound read-only on
top of a write grant. The process runs as your own uid, so files the agent
creates are yours.

Before executing the agent, Vitrine applies a Landlock ruleset that permits
writes only under the write paths, duplicating what the mounts enforce. A
small seccomp filter denies the `TIOCSTI` ioctl, which would otherwise let a
sandboxed process inject keystrokes into your terminal. PID, IPC and UTS
namespaces are unshared and the sandbox dies with the shim.

## 5. Proving it

`vitrine doctor` builds a throwaway tree, grants it, and runs `vitrine probe`
inside the real sandbox. The probe attempts a fixed list of operations, from
reading a credential file to writing `.git/hooks`, and reports each as allowed
or denied. Doctor exits non-zero if anything that must be denied was allowed.
The integration tests run the same probe, so the list is the single source of
truth for what the sandbox guarantees.
