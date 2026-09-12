# Platforms

Vitrine requires Go 1.24 to build. There are no runtime dependencies beyond
the operating system tools listed here.

## macOS

Tested on Apple Silicon with macOS 14 and later.

### Setup

```sh
go install github.com/nimishgj/vitrine/cmd/vitrine@latest
sudo vitrine init
export PATH="$HOME/.vitrine/bin:$PATH"   # add to your shell profile
vitrine doctor
```

`sudo vitrine init` does four things and nothing else:

- creates a hidden system user `vitrine` (uid below 500, no login window
  entry) with home `/var/lib/vitrine`, owned by that user
- creates `/var/lib/vitrine/sessions` with mode 1777, like `/tmp`, so your
  user can create a directory per session there
- writes `/etc/sudoers.d/vitrine`, validated with `visudo`, allowing your user
  to run exactly `<path-to-vitrine> launch *` as `vitrine` without a password
- writes a `.gitconfig` for the `vitrine` user marking all directories safe,
  so agents can run git in repositories you own

The sudoers rule names the absolute path of the binary. Moving or reinstalling
Vitrine requires running `sudo vitrine init` again; `vitrine status` reports
when the rule does not cover the current binary.

### How isolation works here

The agent runs as the `vitrine` user under a per-session Seatbelt profile.
See [how it works](how-it-works.md) for the layers. Practical consequences:

- Files the agent creates in a write grant are owned by `vitrine`. The grant's
  inherited access control entry keeps them editable and deletable by you.
- Your home is mode 700. Grants add a search-only entry on each ancestor so
  the `vitrine` user can reach the granted directory and nothing beside it.
- Agent installs under your home (for example `~/.local/share/claude`) get a
  read-only entry at launch so the sandbox can execute them. This is recorded
  as `binary_root` in the session's audit event.
- Terminal access works through the inherited pty; the profile allows the
  terminal ioctls agents need for raw mode and window size.
- Privacy-protected folders (Desktop, Documents, Downloads) may trigger a
  macOS permission prompt the first time the `vitrine` user touches them.

### Uninstall

```sh
sudo rm /etc/sudoers.d/vitrine
sudo dscl . -delete /Users/vitrine
sudo dscl . -delete /Groups/vitrine
sudo rm -rf /var/lib/vitrine
vitrine grants          # revoke each grant first to remove its ACL entries
rm -rf ~/.vitrine
```

## Linux

Tested on Ubuntu 24.04 in CI.

### Setup

```sh
sudo apt install bubblewrap        # or the equivalent package
go install github.com/nimishgj/vitrine/cmd/vitrine@latest
vitrine init
export PATH="$HOME/.vitrine/bin:$PATH"
vitrine doctor
```

No root is needed for `vitrine init` on Linux; there is no dedicated user
because the mount namespace is the boundary and the agent runs as your uid.

### Prerequisites

- **bubblewrap** (`bwrap`) on `PATH`.
- **Unprivileged user namespaces.** Most distributions enable them. Ubuntu
  24.04 and later restrict them through AppArmor; either install an AppArmor
  profile for `bwrap` (newer Ubuntu releases ship one) or run
  `sudo sysctl kernel.apparmor_restrict_unprivileged_userns=0`.
  `vitrine doctor` prints the exact fix it needs.
- **Landlock** (kernel 5.13 or later) for the second enforcement layer. If it
  is missing Vitrine still runs, because the namespace is the primary
  boundary, and `vitrine status` reports it.

### How isolation works here

bubblewrap constructs a fresh mount namespace per session. Your home path
exists inside it as an empty tmpfs, so nothing under it is visible except the
paths bound in for the grant, the agent's state home and credentials. `/tmp`
is a private tmpfs. `.git/hooks` and `.git/config` are re-bound read-only
inside a write grant. Landlock then restricts writes to the same set, and a
seccomp filter blocks the `TIOCSTI` ioctl so the agent cannot type into your
terminal after it exits. The network namespace is shared, so outbound
connections work.

## Verifying either platform

```sh
vitrine doctor
```

Every row must read `ok`. If a row that must be denied reads `allowed`, do not
run agents until it is fixed; that is the sandbox telling you it has a hole.
The same checks run in the integration test suite:

```sh
go test -tags integration ./...                 # Linux
VITRINE_BIN=$(which vitrine) go test -tags integration ./...   # macOS, after sudo vitrine init
```
