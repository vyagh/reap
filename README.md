# reap

Nothing in Claude Code deletes a transcript. Press ctrl+x on a chat in the agents view and it drops the job record, but the transcript is still on disk and the chat still shows up in `/resume`. There's no delete in `/resume` either, and none on the CLI.

`reap` is a small terminal tool that removes a chat for real: the transcript, its sidecar directory, and the job record, together. One Python file, standard library only. Deletes go to a trash you can undo from for 7 days.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/vyagh/reap/main/reap -o ~/.local/bin/reap
chmod +x ~/.local/bin/reap
```

`~/.local/bin` needs to be on your `PATH`. Python 3.8 or newer.

## Try it before you trust it

`reap rm` is a dry run unless you add `--apply`, and it refuses anything that's still running:

```
$ reap rm 85b3
would trash 85b3f4fe-cb46-47f4-9651-65619d2b40c4 (transcript + sidecar + job)
dry-run, add --apply to trash (--hard to skip trash)

$ reap rm 7d0a
!! SKIP 7d0a6d37-f7a6-412b-a7cc-d5f6ae5fdb51: session is live
nothing to do
```

It only ever writes to `~/.claude/projects/`, `~/.claude/jobs/`, `~/.claude/file-history/`, `~/.claude/session-env/`, `~/.claude/tasks/`, and its own trash at `~/.claude/.reap-trash/`, and it skips the `memory/` folders inside projects. To tell what's live it reads `~/.claude/sessions/` and the daemon roster. It never opens settings, credentials, or anything else under `~/.claude`.

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

`reap rm` options: `--apply` to actually delete, `--hard` to skip the trash, `--orphans` to remove leftover sidecar dirs that have no transcript, `--all` with `--orphans` to sweep every project, `-p <name>` to target a project other than the current directory's.

Scratchpad sessions under `/tmp` are hidden by default. `--tmp` shows them.

## Undo

A delete moves the chat to `~/.claude/.reap-trash/` and keeps it there for 7 days. Pruning happens on the next launch, so there's no background process and the disk is freed only when the trash is pruned or emptied.

In the picker, `u` undoes the last delete and `t` opens the trash to restore or purge. On the CLI it's `reap restore <id|name>`, `reap trash`, and `reap trash --empty`. `reap rm --hard` skips the trash when you really do want it gone now.

## The picker

Run `reap` in a project, or `reap --all` for everything grouped by workspace. Move with the arrows or `j`/`k`, `space` to pick, `p` to peek at the conversation without changing anything, `d` to delete what you picked (it asks first). `/` filters by title, id or workspace, `s` flips the sort between recency and size, `z`/`Z` fold groups, `[`/`]` jump between workspaces. `?` lists all of them. On a wide terminal there's a detail pane with the chat's path, size, message count, and a preview of the conversation.

## How it works

A Claude Code chat is three things on disk:

- transcript: `~/.claude/projects/<project>/<id>.jsonl`
- sidecar dir: `~/.claude/projects/<project>/<id>/`, cached tool results and sub-agent logs, usually where the disk space actually goes
- job record: `~/.claude/jobs/<short-id>/`, what the agents view lists

The agents view only manages job records. `/resume` reads the transcript files directly. That's why a chat you removed there keeps coming back, and why `reap` has to remove all three.

A session also leaves edit snapshots under `~/.claude/file-history/<id>/` and small state under `~/.claude/session-env/<id>/` and `~/.claude/tasks/<id>/`. `reap` trashes those with the chat too, so they come back on restore.

The prompts you typed also land in `~/.claude/history.jsonl`, one shared append-only file behind the prompt history. `reap` never touches it: every session writes there, so rewriting it is the one place a bug could damage chats you didn't delete.

A chat counts as live while a process holds it: `reap` checks the pid recorded in `~/.claude/sessions/` and the daemon roster. Exit a session and it's deletable right away. The flip side is that a background job parked with no process running isn't protected, so if you want to keep it, don't delete it.

## License

MIT, see [LICENSE](LICENSE).
