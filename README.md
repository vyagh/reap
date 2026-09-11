# reap

[![version](https://img.shields.io/github/v/tag/vyagh/reap?color=6c6f9c&label=version)](https://github.com/vyagh/reap/tags)

Delete Claude Code sessions for real.

Nothing in Claude Code deletes a transcript. ctrl+x in the agents view drops the job record, but the transcript stays on disk and the chat is back in `/resume`.

`reap` removes the whole session. Dry run by default, live sessions refused, and a trash you can undo from.

![reap: pick two chats, delete them to the trash, open the trash, undo](docs/reap.gif)

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/vyagh/reap/v0.4.0/reap -o ~/.local/bin/reap
chmod +x ~/.local/bin/reap
```

Python 3.8 or newer, nothing else. Linux and macOS, or WSL on Windows.

## Use

```
reap            picker for this project
reap --all      picker for every project, grouped by workspace
```

| key | does |
|---|---|
| `space` | pick |
| `d` | delete what you picked, after a confirm |
| `u` | undo what you just deleted |
| `p` | read the chat first |
| `/` | filter |
| `t` | open the trash |
| `?` | all keys |

For scripts, the same without the UI:

```
reap ls --all           list, pipe-friendly
reap rm 85b3            dry run: says what it would trash
reap rm 85b3 --apply    trash it
reap restore 85b3       bring it back
reap trash --empty      free the disk now
```

## Safety

**Dry run by default.** `rm` only says what it would do until you add `--apply`:

```
$ reap rm 85b3
would trash 85b3f4fe-cb46-47f4-9651-65619d2b40c4 (transcript + sidecar + job)
dry-run, add --apply to trash (--hard to skip trash)
```

**Live sessions are refused**, in the picker and on the CLI:

```
$ reap rm 7d0a
!! SKIP 7d0a6d37-f7a6-412b-a7cc-d5f6ae5fdb51: session is live
nothing to do
```

**Deletes go to a trash** at `~/.claude/.reap-trash/`, kept 7 days and pruned the next time reap runs. `u` in the picker undoes what you deleted in that run; `t` or `reap restore` brings back anything older. `--hard` skips the trash.

**It writes only under** `~/.claude/projects/`, `jobs/`, `file-history/`, `session-env/`, `tasks/` and its own trash. Settings, credentials, `memory/` folders and `history.jsonl` are never touched.

## How it works

A chat is several things on disk, and the built-in views each touch one:

```
~/.claude/
  projects/<project>/
    <id>.jsonl        transcript          what /resume reads
    <id>/             sidecar dir         tool results and sub-agent logs
  jobs/<short-id>/    job record          what the agents view lists, ctrl+x removes only this
  file-history/<id>/  edit snapshots, often the biggest of all
  session-env/<id>/   small state
  tasks/<id>/         small state
```

That's why a chat removed from the agents view keeps coming back. `reap` moves all of it to the trash together, and back on restore.

A session is live while a process holds it. `reap` checks the pid recorded in `~/.claude/sessions/` and the daemon roster. Exit a session and it's deletable right away, on the machine that ran it. A background job with no process isn't protected.

| option | does |
|---|---|
| `--orphans` | remove sidecar dirs that lost their transcript |
| `--orphans --all` | the same across every project |
| `rm -p <name>` | delete in another project; the picker and `ls` take the name directly |
| `--tmp` | show scratchpad sessions under `/tmp` in the picker and `ls` |

## License

MIT, see [LICENSE](LICENSE).
