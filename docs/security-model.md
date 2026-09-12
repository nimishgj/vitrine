# Security model

## The claim

An agent run through Vitrine cannot modify anything you did not explicitly
grant, cannot read your credentials, and leaves a record of every session and
every grant that cannot be altered without detection.

That claim rests on operating system boundaries, not on the agent. Nothing
the agent does, including running with its own permission checks disabled,
changes what the sandbox allows.

## Two tiers of guarantee

**Remote systems** (Kubernetes, databases, cloud accounts) are read-only with
no exceptions. The identity the agent holds is minted by a provider with
read-only rights inside the target system itself. Even if the sandbox were
bypassed, that identity cannot write. Providers are the next sub-projects;
the interface exists today.

**Local files** are read-only by default with explicit, audited exceptions.
A grant is a deliberate act: you answered a prompt or ran `vitrine grant`.
Every grant is recorded and visible in `vitrine grants` and `vitrine audit`.

## What a write grant really means

If the agent can write to a repository, it can edit a Makefile, a
`package.json` script or a `.envrc`. You later run `make` or `npm test` as
yourself, and that code runs with your full privileges. A write grant on a
project is therefore a delayed grant of your identity for that project.

Vitrine reduces the most direct paths: `.git/hooks` and `.git/config` inside
a write grant are always read-only, because both make git run arbitrary
commands. It refuses grants on your home directory itself, on any
dot-directory directly under it (`.ssh`, `.kube`, `.aws`, `.config`, `.npmrc`
and so on), on system roots and on Vitrine's own state. It cannot enumerate
every file that some tool will later execute. Grant writes to code you will
review.

## Tamper evidence, not tamper prevention

You own your laptop and every file Vitrine writes there. Vitrine does not try
to stop you editing them; it makes every edit attributable.

The audit log is a hash chain: each event carries the hash of the previous
one. Editing, deleting or truncating a line breaks the chain and
`vitrine audit verify` reports where. Every grant and config write records the
resulting file hash in the chain, and every launch checks the files against
it. A change made outside Vitrine produces a tamper event with the diff and
the file's modification time. Grants added that way are not honoured until
`vitrine grants accept` records them, so the acceptance is on record too.

Locally, a determined user could rewrite the whole chain. The format is
designed so that shipping events to a central server, which retains the chain
head, turns attribution into proof. That server is a later tier.

## Known limits of this version

- **Network egress is open.** The agent must reach its model API and any
  remote targets. No allowlist is enforced yet. An agent with read access and
  open egress can still send data out; the guarantee is against mutation, not
  exfiltration.
- **Enforcement is unmanaged.** Nothing stops an engineer running the real
  `claude` binary directly. Vitrine is a boundary you opt into per launch.
  Managed enforcement through endpoint policy is a later sub-project, and
  `vitrine status` reports the level honestly.
- **macOS relies on `sandbox-exec`**, which Apple marks deprecated but has
  kept unchanged for years and depends on internally. The `vitrine` user is
  the second layer if a profile ever has a hole.
- **Shared temp is writable on macOS.** Agents hardcode `/tmp` paths. Files
  there are protected by ordinary per-user permissions; your own temp files
  remain unreadable.
- **Reading is not always safe.** A read-only identity can still read a
  secrets store or a users table. Choosing what a read-only identity may see
  belongs to each provider's setup.

## Reporting a bypass

If you find an operation the sandbox should deny and does not, the fix always
includes a new row in the probe list, so `vitrine doctor` catches it from
then on.
