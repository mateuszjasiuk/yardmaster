# Using ym

Keys below are the defaults; all of them can be changed in [config.md](config.md#keys).

## The status bar

A pinned status bar sits at the top at all times. It is a row of spaced **Nerd Font** glyphs (a
Nerd Font is required). The yardmaster is a seedling, lit cyan while you tend the deck and dim
once you drop into a session; the nvim (code) and term (terminal) icons glow blue and the agent
(robot) purple when active, and dim otherwise (faint until a workspace exists). When an agent is
waiting on your input, the `! N` badge on the right signals it. The active repo / worktree shows
on the right. You can **click** any of the four icons to jump to that view (the session icons do
nothing until a workspace is active).

At startup the deck shows `⠹ scanning repos…` until git has looked at every worktree. ym then
reopens the worktrees you had open, and the bar shows `⠹ restoring 3 of 14…` until each one's
nvim, agents and terminals have drawn; then it says `restored 14 workspace(s)`.

## The deck

![The ym deck showing repositories, active worktrees, branch divergence, and the ambient plasma panel](screenshots/worktree-deck.png)

The deck (seedling) has a `NEW` repo fuzzy-finder and an `ACTIVE` list of your worktrees grouped
by repo (`●` running · `○` stopped · `✷` uncommitted changes).

- Within each repo, worktrees are ordered by when you last opened them in ym (most recent first),
  falling back to git commit time for ones you haven't opened yet. The three most recently opened
  worktrees overall get a `1`/`2`/`3` rank in the left column.
- Before the worktree name, each row shows its session name: the active agent, and `+N` when the
  worktree has more agents (`Abra +1`). Stopped worktrees show the agents they will resume. The
  column is left out when the deck is too narrow.
- Each row shows how far its branch is ahead/behind the repo's main branch (`↑N ↓M`,
  right-aligned). The selected row expands a `└` detail line with new context only: every agent's
  name (when there is more than one), uncommitted diffstat, last commit subject, age, and (when
  available) pull-request state.
- Last-opened times are saved in `$XDG_STATE_HOME/ym/state.json` (default
  `~/.local/state/ym/state.json`).
- A repo's main checkout, the `(main)` row, is an ordinary worktree: open it to get a plain agent
  on the main branch. The row is first in its repo. A repo with no other worktree shows it only
  once it has been opened: type `main` as the name of a new session, or use "open main of <repo>"
  in the command palette.

Deck keys: `tab` switch section · `enter` open · `d` stop · `v` view diff · `x` remove · `r`
refresh · `?` help · `ctrl+c` quit. You can also **click** a row to select it and click it again
to open it (or, in `NEW`, to start naming a worktree).

**Remove.** `x` opens a centered panel with the worktree's divergence and uncommitted diffstat (a
red warning when the tree is dirty) above a list of options: `remove worktree, keep branch` (the
safe default the cursor starts on), `remove worktree and delete branch` (red), `view diff first`,
and `cancel`. Arrow keys (or `j`/`k`) move and `enter` fires the option; the mnemonics also work
directly (`x` `b` keeps the branch, `x` `y` deletes it). The quit and stop guards use the same
panel, starting on `cancel`.

**Diff.** `v` opens a read-only diff of everything the branch carries beyond main (committed and
uncommitted; `u` narrows it to the uncommitted changes). It is also offered from the remove panel,
and `x` from the diff goes back to the panel. See `[diff]` in [config.md](config.md) for git flags,
exclusions and an external pager such as `delta`.

## The workspace

![A workspace with three terminal panes running tests, a development server, and a process monitor](screenshots/split-terminal-workspace.png)

Pressing **enter** on a worktree starts nvim, an agent from `agents.default` and a terminal in
that worktree, and drops you into the nvim view. Sessions keep running: switching never relaunches
them, and they last for ym's lifetime.

A pane whose program has not drawn yet shows `⠹ starting claude…` in its middle (after 150 ms, so
a fast start shows nothing). The seconds appear after 5 s, and after 30 s it says `still
starting` and how to restart the agent. Keys you type into a starting agent wait and are sent
once it has drawn (`3 keys waiting`); `esc` or `ctrl+c` drops them and goes through at once. A
program that exits before it draws anything leaves a red `claude exited before it started`.

- `alt+]` / `alt+[` cycle the views (nvim → agent → terminal → nvim), and `alt+1` / `alt+2` /
  `alt+3` jump straight to one.
- `ctrl+g` returns to the deck. From the deck, `ctrl+g` jumps to your most recently opened
  worktree, so it toggles between the deck and your latest work.
- **Terminal panes:** `alt+v` / `alt+s` split right / below, `alt+z` zooms, `alt+x` closes,
  `alt+h/j/k/l` move focus. `shift+pgup` / `shift+pgdown` walk a pane's scrollback.
- **Side by side:** `ctrl+|` (`ctrl+shift+\`) shows nvim on the left and the focused agent on the
  right, half the width each. `alt+1` / `alt+2`, `alt+h` / `alt+l` or a click move the keyboard
  between them; the side without focus dims (`display.side_by_side_dim`), and the bar lights both
  icons. Cycling treats the two as one stop (side by side → terminal → side by side). Side by side
  needs 121 columns; without an agent nvim fills the width. ym remembers the choice. Your terminal
  has to report `ctrl+|` as its own key (the kitty keyboard protocol); without it the key arrives
  as `ctrl+\`, so pick another `side_by_side` key.
- **Session list:** `ctrl+\` opens the bar's open worktrees on the left, in the same order, under
  their repo, and under their feature (`⚑ name`, see [ctl.md](ctl.md)) or gh stack. A feature's
  manager comes first, marked `★`. Each agent session is a box: its sign (`!` waiting, `✓` done, `●`
  working, `…` starting) and name in the top border, its worktree inside. A box with a sign has its border in
  the sign's colour. The session you'd come back to (the current worktree's active agent) has a
  heavy border and an accent bar on its left, in the accent colour when it has no sign. A stage the
  session set with `ym ctl stage` (`step 2`, `review`) sits in the box's bottom border. Click a
  box to open that session. When the boxes don't fit, the list scrolls with the mouse wheel and
  always moves to show the active session when it changes. `↑ N more` and `↓ N more` count the
  boxes out of view, with how many of them wait for you (`!`) or are done (`✓`); a click on one
  scrolls a page. The list pushes the session to the right (28 columns) instead of covering it, and the bar
  hides its strip while it shows. It hides in terminals under 89 columns (under 150 while side by
  side is on) and on the deck. ym remembers whether it is open. When the bar's strip runs out of
  room, its `+N` opens the list too.
- `alt+g` opens the home-directory workspace (`~`).

## The agent strip

The bar lists every open worktree after its icons, in a fixed order, coloured by what its agents
are doing: yellow while working, green `✓` when done and unread, red `!` when waiting for you (or
exited before it started), `…` while starting, and dim when idle. White means "look at this": a finished agent you have looked at turns white ("your
turn" in the board) until it works again. The current worktree is underlined.

In each repo, the main checkout comes first, unless it belongs to a feature. Then come the
worktrees of a feature (the manager, then the members, a gh stack's members in stack order), then
the repo's other worktrees by name. Inside a feature, a gh stack is neither grouped nor folded.

- Click a name to jump there, or press `ctrl+h` / `ctrl+l` to move left / right like tabs
  (wrapping). ym takes these two keys from the programs inside it, such as nvim and the shell.
- With more than one repo open, each repo's worktrees follow its name
  (`front: beta · zeta │ back: alpha`); the repo names are hidden first when space is short.
- **Flags:** `ctrl+q` flags the current worktree with a colour, so you can mark the ones you want
  to come back to. Each flag takes the next free colour from `display.highlight_colors` (purple,
  cyan, pink, teal by default) and starts again from the first when all are taken. Press `ctrl+q`
  again to remove the flag. While an agent is working, done or waiting, its status colour shows
  instead; the board always shows the flag colour. On a white worktree `ctrl+q` dismisses its "your
  turn" and makes it dim.
- **Attention:** `alt+n` (from anywhere) jumps into the agent that most needs you: waiting on input
  first, then unread completions. Pressing it again cycles to the next one. Clicking the `! N`
  badge does the same. `alt+b` returns to where you were before the last jump, restoring its screen
  and focused pane; press it again to toggle back.
- Branches in a `gh stack` are grouped; see [stacks.md](stacks.md#gh-stack-in-the-bar).

## The agent board

![The agent board showing Claude and Codex agents across multiple worktrees](screenshots/agent-board.png)

`alt+a` opens the board for every active worktree. It opens on a search: type part of a repo,
worktree or agent name (`885` finds `sch-885-…`), and the best match moves to the top, selected.
Worktrees without a running agent match too, listed after the agents: `enter` opens them like the
deck does. `enter` on an agent focuses it, and `esc` closes the board.

`ctrl+e` creates an agent, `ctrl+r` restarts the selected one in place, `ctrl+d` removes it, and
`ctrl+x` dismisses its "your turn". These four are chords because the search takes every letter;
change them with the `board_*` keys. `ctrl+j` / `ctrl+k` move the selection down / up like the
arrows (`board_down` / `board_up`). `f3` / `f4` move between the agents of the current worktree.

The new-agent form passes its prompt as the CLI's first prompt; agents run interactively. The
prompt supports several lines: `ctrl+enter` launches, `enter` adds a line. When more than one
provider is enabled, the form adds a Claude/Codex selector, starting on `agents.default`. Board
and bar labels include the provider.

Every new agent gets a name no other session uses, by default a first-generation Pokémon
(`agent_names` in [config.md](config.md)). Claude gets it as its session name (`claude -n`).

## Command palette, help and usage

![The searchable ym command palette with live keyboard shortcuts](screenshots/command-palette.png)

- `alt+p` (anywhere) opens the **command palette**: a fuzzy-searchable list of every ym action
  with its current key, so you can run any action and learn its key. `enter` runs, `esc` closes.
- `f1` (anywhere) or `?` (in the deck) shows the key and legend overlay; any key closes it.
- `alt+u` shows plan usage for every enabled provider. The gauge in the bar follows the focused
  agent, so a Claude session never shows Codex usage and the other way round.

## How agents are tracked

- Claude status comes from `claude agents --json`. A Claude agent shows its session name from
  Claude, so a `/rename` there shows on the board within one status poll and is saved as the label.
- Each Codex pane owns a private `codex app-server` on a local Unix socket. The stock Codex TUI
  connects to it with `--remote` and still handles approvals and input; ym's passive connection
  watches thread, turn, waiting, completion, failure and disconnect events. This interface is
  experimental, which is why the Codex version matters. Codex events feed the same busy, waiting,
  attention and bell system as Claude, and a failing Claude poll does not erase Codex status.
- Agent state (provider, conversation id, label, pool order, focused agent) is saved in
  `$XDG_STATE_HOME/ym/state.json`. Fresh Claude sessions get a ym-made id right away; Codex thread
  ids are taken from the App Server when the thread starts. On the next open, ym resumes those
  conversations (`codex resume <thread-id>` for Codex). Older state without a provider counts as
  Claude.
- Restart keeps the provider, conversation, label, position and focus. ym starts the replacement
  before swapping it in, so a bad command leaves the current process running.

## Keys

| Key | What it does |
|---|---|
| `ctrl+g` | deck (from the deck: latest worktree) |
| `alt+1` / `alt+2` / `alt+3` | nvim / agent / terminal view |
| `alt+]` / `alt+[` | next / previous view |
| `ctrl+shift+\` | nvim and the agent side by side |
| `ctrl+\` | session list |
| `ctrl+h` / `ctrl+l` | previous / next worktree |
| `ctrl+y` | collapse or expand gh stacks in the bar |
| `ctrl+s` | current gh stack's PRs, reviews and checks |
| `ctrl+q` | flag the worktree with a colour |
| `alt+n` | agent that needs you (repeat to cycle) |
| `alt+b` | back to the previous place |
| `alt+a` | agent board |
| `ctrl+e` / `ctrl+r` / `ctrl+d` / `ctrl+x` | in the board: new / restart / remove / dismiss |
| `f3` / `f4` | previous / next agent in the worktree |
| `alt+v` / `alt+s` | split the terminal right / below |
| `alt+z` / `alt+x` | zoom / close the terminal pane |
| `alt+h/j/k/l` | move focus between panes |
| `shift+pgup` / `shift+pgdown` | scroll a pane |
| `alt+g` | home-directory workspace |
| `alt+u` | plan usage |
| `alt+p` | command palette |
| `f1` | key overlay |
