# Grants and the audit log

## Grants

A grant allows agents to read, or read and write, under one directory. Grants
live in `~/.vitrine/grants.json`, owned by you and invisible inside the
sandbox.

```sh
vitrine grant ~/code/myproject             # read and write
vitrine grant ~/code/reference --read-only # read only
vitrine grants                             # list
vitrine revoke ~/code/reference
```

### Matching

The grant that applies is the nearest ancestor of the directory you launch
from. A read-only grant nested inside a write grant narrows it for that
subtree. A path is only covered by a grant that is an ancestor or exact match;
`~/code/myproject` does not cover `~/code/myproject-old`.

Paths are canonical: symlinks are resolved when the grant is created and when
the session starts, so a symlink cannot route around a rule.

### The first-run prompt

Launching an agent in a directory without a grant asks once:

```
vitrine: claude wants to run in /Users/you/code/myproject
  [w] allow writes here    [r] read-only    [n] cancel
```

The answer becomes a grant with source `prompt`. It is not asked again for
that directory or anything under it. Without a terminal, for example in a
script, there is no prompt; the launch fails and tells you to run
`vitrine grant`.

### Refused paths

`vitrine grant` refuses, in either mode:

- the filesystem root and system roots such as `/usr` and `/System`
- your home directory itself
- any path directly under home whose name starts with a dot: `.ssh`, `.kube`,
  `.aws`, `.gnupg`, `.config`, `.docker`, `.npmrc`, `.netrc` and the rest
- anything under `~/.vitrine`

Dot-directories deeper in the tree, such as `~/code/project/.git`, are fine.

### Inside a write grant

`.git/hooks` and `.git/config` are always read-only, because both can make
git execute arbitrary commands the next time you run it.

### macOS access control entries

Your home directory is mode 700, so the `vitrine` user cannot even traverse
into it. A grant therefore adds access control entries: search-only on each
ancestor directory down from home, and read or read-write with inheritance on
the granted directory. A write grant also adds an inherited entry for you, so
files the agent creates remain yours to edit and delete. `vitrine revoke`
removes exactly these entries and keeps ancestor entries that another grant
still needs. Use `ls -le` to inspect them.

## The audit log

`~/.vitrine/audit.jsonl` holds one JSON object per line:

```json
{"seq":12,"ts":"2026-09-12T10:00:00Z","prev":"<hash of line 11>","type":"session.start","data":{...},"hash":"<hash of this line>"}
```

| Event | Recorded data |
|---|---|
| `init` | platform, version, binary path |
| `config.write` | hash of the config file after the write |
| `grant.add`, `grant.revoke`, `grant.accept` | path, mode, source, hash of the grants file, full grants snapshot |
| `session.start` | session id, agent, binary and install root, directory, grant path and mode, identity minted per target, backend |
| `session.end` | session id, exit code, duration |
| `tamper` | which file, expected and actual hash, modification time, diff of added, removed and changed grants |
| `doctor` | backend, pass or fail, every probe check |

```sh
vitrine audit               # human-readable, oldest first
vitrine audit --since 24h
vitrine audit --json        # raw lines
vitrine audit verify        # exit 3 if the chain is broken
```

### Verification

Each line's `hash` is the SHA-256 of the line without that field, and `prev`
is the previous line's hash. `~/.vitrine/audit.head` holds the latest hash so
a truncated tail is detected as well. `vitrine audit verify` recomputes the
whole chain and names the first sequence number that does not match.

### Tamper detection at launch

Before every session Vitrine hashes `grants.json` and `config.toml` and
compares them with the last hashes recorded in the chain. If either differs:

- a `tamper` event is appended with the diff and the file's modification time
- for grants, only entries that are both in the file and in the last recognised
  snapshot are honoured, at the narrower of the two modes; entries added
  outside Vitrine are ignored and a warning names them
- for config, the file is used but the event stands

`vitrine grants` marks unrecognised entries. `vitrine grants accept`
re-validates every entry against the refusal list, applies platform ACLs, and
records the whole set as recognised. Nothing is silently adopted.
