# yardmaster

A terminal deck for git worktrees. Each task gets its own worktree with nvim, one or more
Claude Code or Codex agents, and a terminal, all hosted *inside* `ym` under a pinned status bar.
You see which agents are working, done or waiting for you, and jump between them.

![The ym deck showing repositories, active worktrees, branch divergence, and the ambient plasma panel](docs/screenshots/worktree-deck.png)

## Requirements

- Go 1.25+, `git`, and your editor (`nvim` by default) on `PATH`
- A [Nerd Font](https://www.nerdfonts.com/) in your terminal
- The CLI of each agent you use, on `PATH` and logged in:
  [Claude Code](https://docs.anthropic.com/en/docs/claude-code) and/or
  [Codex](https://github.com/openai/codex) (`codex-cli` 0.144.4 or newer, because ym uses its
  experimental App Server interface)

## Install

From a checkout:

```sh
go install ./cmd/ym
ym version
```

There is no release yet. Once there is one, macOS binaries and an install script will be
available; see [docs/releasing.md](docs/releasing.md).

## Configure

`ym` reads `~/.yardmaster/config.toml`, or the path in `YM_CONFIG`. Only a repo root is required:

```toml
root = "~/code"                 # parent folder of your repos (or roots = [...] for several)

# Optional:
worktree_path = ".worktrees/{name}"
restore_sessions = true         # reopen last time's worktrees on start

[worktree_files]                # files git ignores, brought into each new worktree
copy = [".env", ".env.local"]

[agents]                        # both are enabled by default
default = "claude"
enabled = ["claude", "codex"]
```

Every key, the key bindings, and several Claude accounts are in [docs/config.md](docs/config.md).

## Main ideas

- **The deck** lists your repos and their worktrees. `enter` opens a worktree, or names a new one in
  the `NEW` section. `v` shows its diff against main, `x` removes it (and asks first).
- **One worktree per task.** Opening one starts nvim, an agent and a terminal there. They keep
  running while you work elsewhere, until you stop them or quit ym.
- **The agent strip** in the top bar lists every open worktree, coloured by its agents: yellow
  working, green `✓` done, red `!` waiting for you. `ctrl+h` / `ctrl+l` move through it like tabs,
  `alt+n` jumps to the agent that needs you, and `ctrl+q` flags a worktree with a colour.
- **The agent board** (`alt+a`) searches every agent and worktree. From there you start, restart or
  remove agents.
- **`ym ctl`** lets agents (and you) start, brief and stop other sessions from the command line.

## Main keys

All keys can be changed in `[keys]`. `alt+p` searches every action and shows its key; `f1` shows
the key overlay.

| Key | What it does |
|---|---|
| `ctrl+g` | back to the deck (from the deck: your latest worktree) |
| `alt+1` / `alt+2` / `alt+3` | nvim / agent / terminal view |
| `alt+]` / `alt+[` | next / previous view |
| `ctrl+shift+\` | nvim and the agent side by side |
| `ctrl+h` / `ctrl+l` | previous / next worktree in the bar |
| `alt+n` | jump to the agent that needs you |
| `alt+b` | back to where you were before the jump |
| `alt+a` | agent board |
| `ctrl+q` | flag the worktree with a colour |
| `ctrl+\` | session list on the left |
| `alt+v` / `alt+s` | split the terminal right / below |
| `alt+p` | command palette |
| `f1` | key overlay |

More on each screen: [docs/usage.md](docs/usage.md).

## ym ctl

`ym ctl` talks to the running ym over a Unix socket. Agents started by ym get `YM_SOCKET`, so they
can run it too:

```sh
ym ctl list                                  # every worktree, its agents and their status
ym ctl spawn myrepo fix-login --prompt "Fix the redirect loop after login"
ym ctl send myrepo/fix-login "Add a test for the expired-token case"
ym ctl summary myrepo/fix-login              # commits, diffstat, last agent message
ym ctl stop myrepo/fix-login                 # close its sessions, keep the worktree
```

Messages wait until the agent is idle. `ym ctl` cannot answer an agent's permission prompt; the
`!` in the bar shows where you need to. All commands, features and their managers:
[docs/ctl.md](docs/ctl.md).

## Stacked PRs

`ym stack status | submit | restack | merge` keeps one GitHub PR per commit of a branch. The bar
also shows branches of a `gh stack` with their position and PR. See [docs/stacks.md](docs/stacks.md).

## Credentials and privacy

`ym` has no login of its own and keeps no copies of provider credentials. To show Claude usage
limits, it reads Claude Code's OAuth credential from the macOS Keychain (or
`~/.claude/.credentials.json`) and sends it only to `https://api.anthropic.com/api/oauth/usage`.
Codex information comes from a local `codex app-server`. There is no telemetry, and no repository
contents, prompts or session output leave your machine through ym.

## More

- [docs/usage.md](docs/usage.md) — every screen and key
- [docs/config.md](docs/config.md) — full configuration
- [docs/ctl.md](docs/ctl.md) — `ym ctl` and feature managers
- [docs/stacks.md](docs/stacks.md) — stacked PRs
- [docs/releasing.md](docs/releasing.md) — install script, releases
- [docs/design-language.md](docs/design-language.md) — how ym should look and behave

Built with [Bubble Tea](https://charm.land/bubbletea) v2 and
[charmbracelet/x/vt](https://github.com/charmbracelet/x).
