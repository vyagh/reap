<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/banner-dark.png">
  <img alt="reap: a chat manager for Claude Code and Codex CLI" src="docs/banner-light.png" width="100%">
</picture>

[![version](https://img.shields.io/github/v/tag/vyagh/reap?color=6c6f9c&label=version)](https://github.com/vyagh/reap/tags)

List, open and delete Claude Code and Codex CLI chats. Deletes are for real.

Claude Code has no way to delete one chat you choose. ctrl+x in the agents view drops the job record. The transcript stays on disk and the chat is back in `/resume`.

- Deletes the whole chat: transcript, side files and job record.
- Claude Code and Codex CLI chats in one list.
- `reap rm` is a dry run until you say `--apply`. The picker asks first.
- A running chat is refused.
- Every delete goes to a trash you can undo from for 7 days, unless you pass `--hard`.

![reap: pick two chats, delete them to the trash, open the trash, undo, then the codex tab](docs/reap.gif)

## Install

Download the file for your system from the [releases page](https://github.com/vyagh/reap/releases). `SHA256SUMS` on that page lists the checksums.

| your system | file |
|---|---|
| Linux, Intel or AMD | `reap-linux-amd64` |
| Linux, ARM | `reap-linux-arm64` |
| Mac, Apple silicon | `reap-darwin-arm64` |
| Mac, Intel | `reap-darwin-amd64` |
| Windows | `reap-windows-amd64.exe` |

```sh
# Linux and macOS: put your file's name in place of reap-linux-amd64
mkdir -p ~/.local/bin
curl -fsSL https://github.com/vyagh/reap/releases/download/v0.6.0/reap-linux-amd64 -o ~/.local/bin/reap
chmod +x ~/.local/bin/reap      # if reap is not found after this, add ~/.local/bin to your PATH
```

```sh
# Windows, in PowerShell, then put reap.exe in a folder on your PATH
Invoke-WebRequest https://github.com/vyagh/reap/releases/download/v0.6.0/reap-windows-amd64.exe -OutFile reap.exe
```

```sh
# with Go 1.27 or newer
go install github.com/vyagh/reap@v0.6.0
```

One program, nothing else to install. Each release is built and tested on Linux, macOS and Windows. On Windows 11 it has been run against a real Claude Code and a real Codex. On macOS it has not been run against either yet.

## Try it

Every step here can be undone. Pick an old chat.

| do | what happens |
|---|---|
| type `reap` | your chats, grouped by project |
| `p` | you read the chat under the cursor. `q` goes back |
| `space`, then `d`, then `y` | the chat is deleted, into the trash |
| `u` | it is back where it was |

`t` shows the trash.

## Use

```sh
reap            # every project, cursor on this folder's project
reap .          # this folder only
reap <name>     # one project, by folder name
```

Enter opens the chat under the cursor in its own agent and folder. With both agents installed, `tab` and `shift-tab` switch between `all / claude / codex`.

<details>
<summary>Keys</summary>

| key | does |
|---|---|
| `enter` | open the chat |
| `space` | pick |
| `a` | pick everything shown, again to unpick. Running chats and folded groups are skipped |
| `d` | delete what you picked, after a confirm |
| `u` | undo what you deleted in this run |
| `p` | read the chat first |
| `h` | hide the chat from the list |
| `.` | show hidden chats, or hide them again |
| `s` | sort: newest, oldest, biggest, smallest |
| `P` | with both agents: make this tab the one reap opens on, again to undo |
| `/` | filter |
| `t` | open the trash |
| `?` | all keys |
| `q` | quit |

</details>

The mouse works too: click a tab, a row or a group line, and the wheel scrolls.

The same from a script:

```sh
reap ls                 # every chat that is not hidden, grouped by project
reap ls --hidden        # the hidden ones
reap rm 5d4a            # dry run, for a chat of this folder
reap rm 5d4a -p <name>  # dry run, for a chat of another project
reap rm 5d4a --apply    # trash it
reap restore 5d4a       # bring it back
reap trash --empty      # purge the trash, no prompt
```

## Safety

One chat from dry run to restore, then a running session that gets refused:

![reap on the command line: dry run, delete to trash, restore, and a live session refused](docs/reap-cli.png)

| what | how |
|---|---|
| Codex chats | deleted through `codex archive`, restored through `codex unarchive` |
| `--hard` | skips the trash. Still a dry run without `--apply`, still refuses running chats |
| never touches | settings, credentials, anything inside a `memory/` folder and the prompt history files |

Where the trash is, what reap writes to, how it tells that a chat is running on each system, and the remaining options are in [docs/details.md](docs/details.md).

## Coming from 0.5.0

- reap is one program now. Replace the old Python `reap` file with it.
- The state file and the trash you made with 0.5.0 are still read.
- A Codex chat's trash entry now lives in `~/.codex/.reap-trash/`. Entries 0.5.0 left in `~/.claude/.reap-trash/` are still listed, restored and pruned.
- A chat that cannot be moved back on restore prints the reason and reap exits 1.

## License

MIT, see [LICENSE](LICENSE).
