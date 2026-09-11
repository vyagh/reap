# reap

[![version](https://img.shields.io/github/v/tag/vyagh/reap?color=6c6f9c&label=version)](https://github.com/vyagh/reap/tags)

Delete Claude Code sessions for real.

Nothing in Claude Code deletes a transcript. Ctrl+x in the agents view drops the job record, but the transcript stays on disk and the chat is back in `/resume`. `reap` removes the whole session, with a dry run, a live-session guard, and a trash you can undo from.

![reap: pick two chats, delete them to the trash, open the trash, undo](docs/reap.gif)

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/vyagh/reap/v0.4.0/reap -o ~/.local/bin/reap
chmod +x ~/.local/bin/reap
```

`~/.local/bin` needs to be on your `PATH`. Python 3.8 or newer, nothing else. Linux and macOS; on Windows use WSL, the picker needs curses.

## Day to day

Open the picker in a project, or across all of them:

```
reap            this project
reap --all      every project, grouped by workspace
```

`space` picks, `d` deletes what you picked (it asks first), `u` undoes, `p` reads a chat before you decide, `/` filters, `t` opens the trash. `?` shows the rest.

The same without the UI, for scripts:

```
reap ls --all           list, pipe-friendly
reap rm 85b3            dry run: says what it would trash
reap rm 85b3 --apply    trash it
reap restore 85b3       bring it back
reap trash --empty      free the disk now
```

## What keeps you safe

`rm` is a dry run unless you add `--apply`:

```
$ reap rm 85b3
would trash 85b3f4fe-cb46-47f4-9651-65619d2b40c4 (transcript + sidecar + job)
dry-run, add --apply to trash (--hard to skip trash)
```

A session with a live process can't be deleted, in the picker or on the CLI:

```
$ reap rm 7d0a
!! SKIP 7d0a6d37-f7a6-412b-a7cc-d5f6ae5fdb51: session is live
nothing to do
```

Deletes go to `~/.claude/.reap-trash/` and stay there for 7 days. Undo with `u` in the picker or `reap restore` on the CLI. `--hard` skips the trash when you mean it.

It writes only under `~/.claude/projects/`, `jobs/`, `file-history/`, `session-env/`, `tasks/` and its own trash. It never touches settings, credentials, the `memory/` folders, or `history.jsonl`.

## How it works

A Claude Code chat is three things on disk, and the built-in views each touch one:

```
~/.claude/
  projects/<project>/
    <id>.jsonl        transcript          what /resume reads
    <id>/             sidecar dir         tool results, sub-agent logs, most of the disk
  jobs/<short-id>/    job record          what the agents view lists, ctrl+x removes only this
  file-history/<id>/  edit snapshots
  session-env/<id>/   small state
  tasks/<id>/         small state
```

That's why a chat you removed from the agents view keeps coming back. `reap` moves all of the above to the trash together, and back on restore.

Live means a process holds the session: `reap` checks the pid in `~/.claude/sessions/` and the daemon roster. Exit a session and it's deletable right away. A background job parked with no process isn't protected.

More options: `--orphans` removes sidecar dirs that lost their transcript (`--all` sweeps every project), `-p <name>` targets another project, `--tmp` includes scratchpad sessions under `/tmp`. `reap --help` has the full list.

## License

MIT, see [LICENSE](LICENSE).
