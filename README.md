# reap

[![version](https://img.shields.io/github/v/tag/vyagh/reap?color=6c6f9c&label=version)](https://github.com/vyagh/reap/tags)

Delete Claude Code sessions for real.

Nothing in Claude Code deletes a transcript. ctrl+x in the agents view drops the job record, but the transcript stays on disk and the chat is back in `/resume`.

`reap` removes the whole session. `rm` is a dry run by default, the picker confirms first, live sessions are refused, and deletes go to a trash you can undo from.

![reap: pick two chats, delete them to the trash, open the trash, undo](docs/reap.gif)

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/vyagh/reap/v0.4.0/reap -o ~/.local/bin/reap
chmod +x ~/.local/bin/reap
```

One file of stdlib Python, so read it before you run it. Python 3.8 or newer. Linux and macOS, or WSL on Windows.

## Use

```
reap            picker for this project
reap --all      picker for every project, grouped by workspace
```

| key | does |
|---|---|
| `space` | pick |
| `d` | delete what you picked, after a confirm |
| `u` | undo the last delete |
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

One chat, start to finish. `rm` is a dry run until you add `--apply`:

```
$ reap rm 86eb
would trash 86eb145e-acc2-42cc-9cf6-d5e58c5aeff9 (transcript, sidecar, file-history, jobs)
dry-run, add --apply to trash (--hard to skip trash)

$ reap rm 86eb --apply
TRASH 86eb145e-acc2-42cc-9cf6-d5e58c5aeff9 (transcript, sidecar, file-history, jobs)
done, in trash for 7d (reap restore to undo)

$ reap restore 86eb
restored 86eb145e  events backfill
```

A session whose process is still running is refused, even with `--apply`:

```
$ reap rm 7d0a --apply
!! SKIP 7d0a6d37-f7a6-412b-a7cc-d5f6ae5fdb51: session is live
nothing to do
```

Deletes go to a trash at `~/.claude/.reap-trash/`, kept 7 days and pruned the next time reap runs. `u` in the picker undoes the last delete, and again for the one before; `t` or `reap restore` brings back anything else. `--hard` skips the trash.

It writes only under `~/.claude/projects/`, `jobs/`, `file-history/`, `session-env/`, `tasks/` and its own trash. Settings, credentials, `memory/` folders and `history.jsonl` are never touched.

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

A session is live while a process holds it. `reap` checks the pid recorded in `~/.claude/sessions/` and the daemon roster. Exit a session and it's deletable right away, on the machine that ran it. A background job with no process isn't protected; it lists like any other chat and can be trashed.

| option | does |
|---|---|
| `rm --orphans` | remove sidecar dirs that lost their transcript |
| `rm --orphans --all` | the same across every project |
| `rm -p <name>` | delete in another project; the picker and `ls` take the name directly |
| `--tmp` | show scratchpad sessions under `/tmp` in the picker and `ls` |

## License

MIT, see [LICENSE](LICENSE).
