# reap: Claude Code chat manager

A small terminal tool to view, peek at, and delete your local Claude Code chats.

Claude Code keeps every conversation as a transcript under `~/.claude/projects/`.
Deleting one from the agents view only dismisses it from the list, the
transcript stays on disk and still shows up in `/resume`. `reap` deletes a chat
for real: its transcript, its sidecar directory, and its agents-view job record,
together. Deletes are reversible for 7 days.

Pure-stdlib Python, no dependencies. It reads and deletes files under
`~/.claude/` and nothing else.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/vyagh/reap/main/reap -o ~/.local/bin/reap
chmod +x ~/.local/bin/reap
```

Make sure `~/.local/bin` is on your `PATH`, then run `reap`.

## Usage

```
reap                     interactive picker for the current project
reap <name|path>         picker for a project (name = folder substring)
reap --all               picker across every project, grouped by workspace
reap ls [<name>|--all]   plain list (no UI; pipe-friendly)
reap rm <id>... [opts]   delete by uuid prefix (for scripts)
reap trash [--empty]     list recoverable deletes (or purge them all)
reap restore <id|name>   restore a chat from the trash
reap --help              full help
```

`reap rm` options: `--apply` (actually delete; default is a dry-run),
`--hard` (skip the trash, delete irreversibly), `--orphans` (remove leftover
sidecar dirs), `--all` (with `--orphans`, sweep every project), `-p <name>`
(target another project).

Ephemeral `/tmp` scratchpad sessions are hidden by default; pass `--tmp` to
include them.

## Delete safely

Delete is a soft delete: the chat is **moved** to `~/.claude/.reap-trash/` and
stays recoverable for 7 days (pruned lazily on launch, no background process).

- In the picker: `u` undoes your last delete; `t` opens a trash browser to
  restore or permanently purge.
- On the CLI: `reap restore <id|name>`, `reap trash`, `reap trash --empty`.
- `reap rm --hard` skips the trash when you really want it gone now.

Live sessions are protected: a chat whose process is still running can't be
selected or deleted. Exit the session and it's immediately deletable.

## Picker keys

| key | action |
| --- | --- |
| `↑`/`↓`, `j`/`k` | move |
| `PgUp`/`PgDn`, `g`/`G` | page / top-bottom |
| `[` `]` | previous / next workspace (in `--all`) |
| `z` / `Z` | fold this group / fold all |
| `space` | select (skips live sessions) |
| `a` / `c` | toggle select-all (filtered) / clear |
| `p` | peek: read-only preview of the chat |
| `d` / `enter` | delete selected → trash (asks to confirm) |
| `u` | undo the last delete |
| `t` | trash browser: restore / purge |
| `s` | sort: recency ↔ size |
| `/` | filter (title / uuid / workspace) |
| `?` | show keys |
| `q` | quit |

On a wide terminal the picker shows a detail pane for the chat under the cursor
with its path, size, message count, and a preview of the conversation.

## How it works

A Claude Code chat is three things on disk:

- **transcript**: `~/.claude/projects/<project>/<id>.jsonl`
- **sidecar dir**: `~/.claude/projects/<project>/<id>/` (cached tool results and
  sub-agent logs; usually where the disk space goes)
- **job record**: `~/.claude/jobs/<short-id>/` (what the agents view lists)

The agents-view delete removes only the job record; `/resume` reads the
transcript directly, so the chat survives. `reap` removes all three together, so
the chat is gone from `/resume` and the agents view, and (once the trash is
pruned or emptied) the disk is reclaimed.

A session counts as "live" only while its process is actually running (checked
by pid via `~/.claude/sessions/` and the daemon roster), so a session you've
exited stops being protected right away.

## Requirements

- Python 3.8+ (standard library only)
- A local Claude Code install (the `~/.claude/` directory)

## License

MIT, see [LICENSE](LICENSE).
