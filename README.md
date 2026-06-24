# reap: claude chat manager

A small terminal tool to view, peek at, and delete your local Claude Code chats.

Claude Code keeps every conversation as a transcript under `~/.claude/projects/`.
Deleting one from the agents view only dismisses it from the list, the
transcript stays on disk and still shows up in `/resume`. `reap` deletes a chat
for real: its transcript, its sidecar directory, and its agents-view job record,
together.

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
reap --help              full help
```

`reap rm` options: `--apply` (actually delete; default is a dry-run),
`--orphans` (remove leftover sidecar dirs), `-p <name>` (target another project).

## Picker keys

| key | action |
| --- | --- |
| `↑`/`↓`, `j`/`k` | move |
| `PgUp`/`PgDn`, `g`/`G` | page / top-bottom |
| `[` `]` | previous / next workspace (in `--all`) |
| `space` | select (skips live sessions) |
| `a` / `c` | select-all (filtered) / clear |
| `p` | peek: read-only preview of the chat |
| `d` / `enter` | delete selected (lists them, asks to confirm) |
| `s` | sort: recency ↔ size |
| `/` | filter (title / uuid / workspace) |
| `?` | show keys |
| `q` | quit |

## How it works

A Claude Code chat is three things on disk:

- **transcript**: `~/.claude/projects/<project>/<id>.jsonl`
- **sidecar dir**: `~/.claude/projects/<project>/<id>/` (cached tool results and
  sub-agent logs; usually where the disk space goes)
- **job record**: `~/.claude/jobs/<short-id>/` (what the agents view lists)

The agents-view delete removes only the job record; `/resume` reads the
transcript directly, so the chat survives. `reap` removes all three together, so
the chat is gone from `/resume` and the agents view, and the disk is reclaimed.

Live sessions are detected (via `~/.claude/sessions/` and the daemon roster) and
can't be selected or deleted.

## Requirements

- Python 3.8+ (standard library only)
- A local Claude Code install (the `~/.claude/` directory)

## License

MIT, see [LICENSE](LICENSE).
