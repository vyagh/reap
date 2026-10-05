# reap in detail

The [README](../README.md) covers install and daily use. This page holds the rest.

## What a chat is on disk

A Claude Code chat is several things on disk, and the built-in views reach only two of them. `CLAUDE_CONFIG_DIR` is honoured, so `~/.claude` below means that folder if you set it:

```
~/.claude/
├── projects/<project>/
│   ├── <id>.jsonl          transcript        what /resume reads
│   └── <id>/               sidecar dir       tool results and sub-agent logs
├── jobs/<short-id>/        job record        what the agents view lists
├── file-history/<id>/      edit snapshots    can be large
├── session-env/<id>/       small state
└── tasks/<id>/             small state
```

ctrl+x in the agents view removes only the job record. That's why the chat keeps coming back. `reap` moves all of it to the trash together, and back on restore. Job records that resume this chat go with it.

A Codex chat is one rollout file under `~/.codex/sessions/` (`CODEX_HOME` is honoured). reap lists the chats under `sessions/` and hands the delete to the `codex` command. After a delete it reads the archived copy in `archived_sessions/` for the trash view.

## Opening a chat

Enter runs `claude --resume <id>` or `codex resume <id>` directly, so shell aliases for `claude` or `codex` do not apply. On Linux and macOS reap quits as the agent starts. On Windows reap waits until the agent exits. `claude` and `codex` have to be on your PATH for this. On Windows the Claude Code installer does not add itself: it prints the folder to add.

Hiding is not deleting. A hidden chat keeps all its files and shows again under `.`. The list of hidden chats and the pinned tab are in `~/.local/state/reap/state.json`.

## When a chat counts as running

Live means a process still holds the session. For Claude Code, `reap` checks the pid in `~/.claude/sessions/` and the daemon roster. For Codex it looks for a held writer lock in `~/.codex/thread-writer-locks/`. On Linux and macOS a held lock counts as live, and so does a lock file reap cannot open. No lock file means exited. On Windows reap does not try the lock: a lock file that exists counts as live.

| the session | what reap does |
|---|---|
| still running | refused, in the picker and on the CLI |
| exited | can be deleted right away, on the machine that ran it |
| a background job with no process | treats it like any other chat |
| started on another machine or in a container | on Linux, kept protected when the session records its machine. Delete it where it ran |
| on macOS | sessions made there are checked by pid only. One that records a Linux machine stays protected |
| on Windows | reap cannot ask the system about a pid, so a Claude chat that has a session file stays protected. Claude Code removes that file when it quits. After a crash it stays until Claude Code is started again, which clears it |
| a session file, or the `sessions` folder, that reap cannot read | every Claude chat counts as running until it is fixed or removed |

## The trash

| what | how |
|---|---|
| where | `~/.claude/.reap-trash/` for Claude chats, `~/.codex/.reap-trash/` for Codex chats. Kept 7 days, pruned the next time reap runs |
| undo | `u` in the picker, most recent first. `t` or `reap restore` for anything older |
| trash screen | `space` picks, `r` restores the picks or the row, `x` purges the picks, `E` empties the tab you are on, or just the filtered rows, `/` filters, `p` reads, `q` goes back |
| Codex chats | reap does not move Codex's files. It runs `codex archive <id>` and the chat waits in `~/.codex/archived_sessions/`. Restore runs `codex unarchive`. After 7 days the next reap command, other than help or version, runs `codex delete --force` on it, no prompt. Restore before then if you want it back |
| `--no-codex-cli` | moves the rollout file into `~/.codex/.reap-trash/` instead of calling `codex`, like a Claude chat. Codex's own record then points at a missing file until you restore |
| no `codex` on PATH | the delete fails with a message and the chat stays where it is. A restore that fails keeps its trash entry so you can retry. A purge still drops the entry and leaves the file in Codex's archive |
| `--hard` | skips the trash. On a Codex id it runs `codex archive` then `codex delete --force`. Still a dry run without `--apply`, still refuses live sessions |

## What reap writes and what it leaves alone

| what | how |
|---|---|
| writes to | `~/.claude/projects/`, `jobs/`, `file-history/`, `session-env/`, `tasks/`, its own trash folders, and `~/.local/state/reap/state.json`. Codex chats only through the `codex` command, unless you pass `--no-codex-cli`. To tell if a Codex chat is running on Linux or macOS, reap opens and briefly locks its file in `~/.codex/thread-writer-locks/` |
| never touches | settings, credentials, anything inside a `memory/` folder (an empty one left alone in a project dir goes with that dir after a purge), and the prompt history files `~/.claude/history.jsonl` and `~/.codex/history.jsonl` |

## More options

| option | does |
|---|---|
| `rm --orphans` | remove sidecar dirs that lost their transcript. Dry run and trash like `rm`. No live check, orphans have no session. `memory/` is skipped |
| `rm --orphans --all` | the same across every project |
| `rm -p <name>` | delete in another project. The picker and `ls` take the name directly. `rm` takes Codex ids as well as Claude ones. A name or `.` needs a Claude Code project folder for that directory. In a folder with only Codex chats, use `rm <id>` there or `-p <path>` |
| `--tmp` | with plain `reap` or `ls`: also show scratchpad sessions under `/tmp` |
