# reap

[![version](https://img.shields.io/github/v/tag/vyagh/reap?color=6c6f9c&label=version)](https://github.com/vyagh/reap/tags)

Delete Claude Code sessions for real.

Nothing in Claude Code deletes a transcript. ctrl+x in the agents view drops the job record. The transcript stays on disk and the chat is back in `/resume`.

`reap` removes the whole session. On the command line it's a dry run until you say `--apply`. The picker asks first. Live sessions are refused, and every delete goes to a trash you can undo from.

![reap: pick two chats, delete them to the trash, open the trash, undo](docs/reap.gif)

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/vyagh/reap/v0.4.0/reap -o ~/.local/bin/reap
chmod +x ~/.local/bin/reap
```

One file of stdlib Python, so read it before you run it. Python 3.8 or newer. Linux and macOS, or WSL on Windows.

## Use

```sh
reap            # picker for this project
reap --all      # every project, grouped by workspace
```

| key | does |
|---|---|
| `space` | pick |
| `d` | delete what you picked, after a confirm |
| `u` | undo what you deleted in this run |
| `p` | read the chat first |
| `/` | filter |
| `t` | open the trash |
| `?` | all keys |

The same from a script:

```sh
reap ls --all           # list every chat, grouped by project
reap rm 5d4a            # dry run: says what it would trash
reap rm 5d4a --apply    # trash it
reap restore 5d4a       # bring it back
reap trash --empty      # purge the trash, no prompt
```

## Safety

One chat from dry run to restore, then a running session that gets refused:

![reap on the command line: dry run, delete to trash, restore, and a live session refused](docs/reap-cli.png)

| what | how |
|---|---|
| trash | `~/.claude/.reap-trash/`. Kept 7 days, pruned the next time reap runs |
| undo | `u` in the picker, most recent first. `t` or `reap restore` for anything older |
| `--hard` | skips the trash. Still a dry run without `--apply`, still refuses live sessions |
| `rm --orphans` | dry run and trash like `rm`. No live check, orphans have no session. `memory/` is skipped |
| writes to | `~/.claude/projects/`, `jobs/`, `file-history/`, `session-env/`, `tasks/`, and its own trash |
| never touches | settings, credentials, `memory/` folders, `history.jsonl` |

## How it works

A chat is several things on disk, and the built-in views reach only two of them:

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

ctrl+x in the agents view removes only the job record. That's why the chat keeps coming back. `reap` moves all of it to the trash together, and back on restore.

Live means a process still holds the session. `reap` checks the pid in `~/.claude/sessions/` and the daemon roster.

| the session | what reap does |
|---|---|
| still running | refused, in the picker and on the CLI |
| exited | can be deleted right away, on the machine that ran it |
| a background job with no process | treats it like any other chat |
| started on another machine or in a container | on Linux, kept protected when the session records its machine. Delete it where it ran |
| on macOS | checks only the pid, there is no machine id |

| option | does |
|---|---|
| `rm --orphans` | remove sidecar dirs that lost their transcript |
| `rm --orphans --all` | the same across every project |
| `rm -p <name>` | delete in another project. The picker and `ls` take the name directly |
| `--tmp` | with `--all`: also show scratchpad sessions under `/tmp` |

## License

MIT, see [LICENSE](LICENSE).
