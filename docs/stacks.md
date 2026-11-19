# Stacked PRs

ym works with two kinds of stacks: its own `ym stack` (one PR per commit of a branch), and
branches you stack with `gh stack`, which ym only shows.

## ym stack

ym treats each commit on a worktree branch as its own GitHub PR, chained bottom to top and
identified by a `ym-stack-id` commit trailer. The `ym stack` CLI drives the workflow from a
worktree:

- `ym stack status` — read-only reconciliation of local commits, last-fetched remote branches, and
  PR state into a per-commit status plus a stack-level `next_action` hint (`--json` for the machine
  form). It never changes anything.
- `ym stack submit` — the additive half: assigns ids to new commits, pushes branches, and creates or
  updates PRs (retarget/retitle/nav-table) to match the local stack. `--dry-run` prints the plan.
- `ym stack restack` — repairs a stack after its bottom PRs squash-land: drops the landed commits
  from the local branch, deletes their remote branches, and re-submits the rest. `--dry-run`
  prints the plan; because it rewrites branches, run the plan first.
- `ym stack merge` — squash-merges the bottom PR with its original commit message. It leaves branch
  cleanup to GitHub's repository setting so dependent stacked PRs are retargeted before the merged
  head branch is deleted. It re-fetches right before merging and refuses unless GitHub reports
  `MERGEABLE` and the PR targets the repository's main branch. After merging, it waits through
  GitHub's short retargeting and mergeability window before publishing the next stack status, so
  short-lived `restack`/`wait` states do not flash in the UI. `[stack] auto_merge = true` skips
  ym's confirmation.

### In the deck

The deck shows stack state without running a subprocess on the render path (a cache is refreshed
after each deck load and on `r`, and shows nothing until data lands or when GitHub is unavailable):

- **Deck glyph** — after a row's `↑N ↓M`: a red `↻` when the stack needs a restack (commits landed
  below), a green `✓` when every commit is open with checks passing, a yellow `…` when any PR's
  checks are pending, and a red `!` for conflicts or escalations (closed PR, duplicate id, broken
  base chain). Nothing shows for an unsubmitted stack.
- **Detail line** — the selected row gains a stack segment: a single-commit stack reads
  `PR #42 open · checks passing`; larger stacks state the outcome, such as
  `1 merged · restack needed`, `resolve conflicts`, or `waiting on checks`.
- **Conflict recovery** — when a PR conflicts after an earlier commit lands, the stack screen keeps
  the conflict visible and offers `R` to preview a restack that drops the landed commits and
  rebases the rest onto current `origin/main`.
- **Merge** — a bottom PR that GitHub reports as mergeable and that targets main shows `M merge` in
  the stack screen and a matching command-palette action. Other PRs never show it.
- **Command palette** — per active worktree: `stack status: <repo>/<wt>` (always),
  `restack: <repo>/<wt>` (only when a restack is needed) and `submit stack: <repo>/<wt>` (only with
  work to submit).

All three open a read-only, scrollable overlay titled `STACK <repo> / <wt>` with the same output as
`ym stack status`. `submit` runs directly (it only adds) and shows the fresh status. `restack`
shows its **dry-run plan first**: `enter` runs it, `esc` cancels without touching your branches.
Inside the overlay, `j`/`k` scroll, `r` re-fetches, and `esc`/`q` close.

## gh stack in the bar

For a branch in a `gh stack`, the bar's label shows its position and PR, such as `4/5 #1366`, and
`ym ctl list` shows `stack=#1369 4/5  pr=#1366`. ym reads this from `.git/gh-stack`; it never
changes a stack.

- The bar's strip keeps a stack's open worktrees together, bottom to top, and collapses each stack
  you're not in to one item (`⛁ #1369`, with its most urgent state). `ctrl+y` shows them all in full
  instead; ym remembers the choice. `ctrl+h` / `ctrl+l` treat a collapsed stack as one stop and
  open the worktree you used last in it.
- The deck lists every stack in full under a `⛁ #1369` header, with each worktree's position and PR.
- `ctrl+s` lists every branch of the current stack (the selected one on the deck) with its PR's
  state, review and checks. ym asks GitHub through `gh` only when the popup opens, at most once a
  minute, or on `r`. `enter` opens the branch's session and `o` its PR.

To add a new session's branch to a stack, see [ctl.md](ctl.md#feature-managers).
