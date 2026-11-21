# Configuration

`ym` reads `~/.yardmaster/config.toml`, or the path in `YM_CONFIG`. Only a repo root (`root` or
`roots`) is required. Below is every key with its default or an example.

```toml
# Parent directory containing your repos (each immediate child with a .git).
root = "~/code"

# Additional trees to discover repos in, for work split across several
# directories. `root` is simply the first of them, and either key satisfies the
# requirement — set only `roots` if you have no single primary tree. Every entry
# must exist at startup. When two trees hold a repo with the same directory name,
# both are shown with enough of their path to tell them apart ("work/api" and
# "personal/api"); repos with a unique name keep the bare name.
roots = ["~/work", "~/personal"]

# Optional program defaults:
editor = "nvim"
shell  = "/bin/zsh" # defaults to the current $SHELL when omitted

# Where new worktrees and their branches are created ({name} is the worktree name).
worktree_path = ".worktrees/{name}"
branch_name   = "{name}"

# Reopen the worktrees that were open when ym last quit. They start in the
# background while the deck is shown.
restore_sessions = false

# Names for new agent sessions. Each new session gets a random name from the
# list that no open session, and no saved session of an existing worktree,
# uses. When all are taken, a number is added ("Pikachu 2"). Leave it out for
# the 151 first-generation Pokémon; set it to [] for random adjective-noun names
# ("amber-fox").
agent_names = ["Ada", "Grace", "Linus"]

# Run a command in the background whenever a session's folder gains focus
# (switching to it, or ym's window getting focus back). The folder is added as
# the last argument. Empty is off.
focus_command = ""

# Files to bring from the repo's main checkout into every new worktree, usually
# ones git ignores. Entries are glob patterns relative to the repo root (`*`,
# `?`, `[...]`; no `**`). `copy` gives each worktree its own version; on macOS
# it is an APFS clone, so even node_modules copies near-instantly and uses no
# extra disk space until a file changes. `link` makes a symlink to the main
# checkout instead, shared by every worktree. A path that already exists in the
# new worktree is never overwritten, and a pattern that matches nothing is
# skipped. If something cannot be copied, ym says so and opens the worktree
# anyway.
[worktree_files]
copy = [".env", ".env.local"]

# Per-repo rules, keyed by the repo name ym shows (quote names like "work/api").
# Each list replaces the default one only when set, so this repo keeps the
# default `link`; set a list to [] to turn it off for one repo.
[repos.api.worktree_files]
copy = [".env*", "apps/*/.env", "node_modules"]

# Both providers are enabled by default, with Claude initially selected, so
# this whole section may be omitted. `default` must also appear in `enabled`.
[agents]
default = "claude"
enabled = ["claude", "codex"]

[agents.claude]
command = "claude"
args = []

[agents.codex]
command = "codex"
args = []

# Ambient plasma panel on the right of the deck (defaults shown). It only
# animates while the deck is on screen, and hides itself on terminals too
# narrow to split.
[plasma]
pattern = "classic" # classic | waves | interference | lava | ripple
palette = "aurora"  # aurora (blue/purple) | ember (yellow/red) | mono (grayscale)
charset = "dots"    # dots (braille) | shade | blocks
speed   = 0.3       # animation rate; 0 freezes the pattern
width   = 40        # percent of the terminal width; 0 disables the panel

# Stack merges require a confirmation by default. This setting bypasses ym's
# confirmation; it is not GitHub's "auto-merge when checks pass" feature.
[stack]
auto_merge = false

# Persistent navigation and pane symbols. `nerd` preserves the default icon
# set; `text` uses labels; `ascii` uses stable single-cell characters.
[display]
icons = "nerd" # nerd | text | ascii
# List every open worktree on its own row under the bar, coloured by its agents.
agent_strip = true
# How the side without focus looks while nvim and the agent are side by side:
# faint keeps its colours darker, gray draws it in plain gray (for terminals
# that don't show faint text).
side_by_side_dim = "faint" # faint | gray
# A rounded box around nvim and the agent, named in its top border. Off brings
# back the line under the bar and the divider between them side by side.
pane_boxes = true
# The colours ctrl+q gives flagged worktrees in the bar, in order ("#RRGGBB").
highlight_colors = ["#BB80B3", "#B2CCD6", "#FF9CAC", "#80CBC4"]

# The diff viewer (deck `v` and the stack `d` preview). All optional.
[diff]
# Extra flags handed to every diff git runs, before the revision range. A later
# flag wins over ym's own, so `--color=always` here feeds a pager that wants
# coloured input. Good ones to try: --histogram, -C, --color-moved=zebra, -W,
# --ignore-blank-lines.
args    = ["--histogram", "-C"]
# Pathspecs dropped from the diff — the body, the file index, and the +/−
# totals all agree. Lockfiles and snapshots are the usual culprits.
exclude = ["*.lock", "go.sum"]

# An external formatter each file's patch is piped through, rendered verbatim
# inside ym's viewer (which keeps scrolling, file jumps, and the scope toggle).
# Omit the whole subtable to keep ym's built-in styling. The program must not
# page itself, and `{width}` is replaced with the viewer's column count.
[diff.pager]
command = "delta"
args    = ["--paging=never", "--width={width}"]
```

## Agent commands

`command` is the executable and `args` is a list of base arguments inserted right after it, so
wrappers work too:

```toml
[agents]
default = "codex"
enabled = ["codex"]

[agents.codex]
command = "mise"
args = ["exec", "--", "codex"]
```

The old top-level `agent = "claude"` setting is still accepted as a Claude command override. New
configurations should use `[agents.claude]` instead.

## Keys

Reserved navigation keys are not forwarded to the programs inside the panes. Defaults use alt
(option-as-meta) chords so they don't collide with LazyVim, zsh, Claude Code or Codex. Every key
can be changed, and an empty string (`""`) turns one off. What each one does is in
[usage.md](usage.md#keys).

```toml
[keys]
cycle      = "alt+]"   # next session view (wraps)
cycle_back = "alt+["   # previous session view (wraps)
goto_editor = "alt+1"  # jump to the editor view
goto_agent  = "alt+2"  # jump to the agent view
goto_term   = "alt+3"  # jump to the terminal view
picker      = "ctrl+g" # return to the deck
palette     = "alt+a"  # agent board: find, focus, create, restart, or remove agents
board_new     = "ctrl+e" # in the board: new agent
board_restart = "ctrl+r" # in the board: restart the selected agent
board_remove  = "ctrl+d" # in the board: remove the selected agent (asks first)
board_dismiss = "ctrl+x" # in the board: dismiss the selected agent's "your turn"
board_up      = "ctrl+k" # in the board: move the selection up (like ↑)
board_down    = "ctrl+j" # in the board: move the selection down (like ↓)
next_agent  = "f4"     # next agent in the active worktree
prev_agent  = "f3"     # previous agent in the active worktree
next_worktree = "ctrl+l" # next worktree in the bar, like tabs (wraps)
prev_worktree = "ctrl+h" # previous worktree in the bar (wraps)
toggle_stacks = "ctrl+y" # collapse or expand gh stacks in the bar
side_panel  = "ctrl+\\" # open or close the session list on the left
side_by_side = "ctrl+shift+\\" # nvim and the agent next to each other (ctrl+|)
stack_popup = "ctrl+s"  # the current gh stack's PRs, reviews and checks
global_config = "alt+g" # open the home-directory workspace
attention   = "alt+n"  # jump to the agent needing attention (cycles on repeat)
highlight   = "ctrl+q" # flag the current worktree with a colour in the bar (toggle)
back        = "alt+b"  # return to the previous work location (toggles on repeat)
usage       = "alt+u"  # plan usage for enabled agent providers
help        = "f1"     # toggle the key/legend overlay (also "?" in the deck)
command_palette = "alt+p" # fuzzy-searchable list of every action + its key

# Pane keys on the terminal screen; focus left / right also moves between nvim
# and the agent while they are side by side.
term_split_v    = "alt+v"  # new pane to the right
term_split_h    = "alt+s"  # new pane below
term_zoom       = "alt+z"  # zoom / restore the focused pane
term_close      = "alt+x"  # close the focused pane
term_focus_left = "alt+h"  # directional pane focus (h/j/k/l)
term_focus_down = "alt+j"
term_focus_up   = "alt+k"
term_focus_right = "alt+l"
# Walk the focused pane's scrollback a screenful at a time; the mouse wheel
# scrolls it by three lines. Typing returns the pane to live output, and the bar
# says so while it is parked. Full-screen programs (nvim, the agent TUIs) keep
# their own scrolling instead.
term_scroll_up   = "shift+pgup"
term_scroll_down = "shift+pgdown"
```

## Several Claude accounts

Claude Code keeps each login in its own config folder, chosen with `CLAUDE_CONFIG_DIR`. To use,
say, a work and a private account side by side, log in to the second one once
(`CLAUDE_CONFIG_DIR=~/.claude-private claude`) and name the folders whose repos should use it:

```toml
[claude_accounts.private]
config_dir = "~/.claude-private"
folders = ["~/Projects"]
```

Claude agents in repos under one of `folders` start with that account's `CLAUDE_CONFIG_DIR`;
every other repo, and the `~` home workspace, uses the default account (`~/.claude`, or whatever
`CLAUDE_CONFIG_DIR` was when ym started). When folders nest, the longer one wins. ym checks at
startup that `config_dir` and every folder exist, and that no two accounts share one.

Status polling covers every account with an open agent, and the bar's agent strip shows an extra
account's worktrees as one block on a darker background, after the default account's. The usage
gauge only reads the default account, so it is hidden next to agents on another one.
