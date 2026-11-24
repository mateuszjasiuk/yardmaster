# ym ctl

`ym ctl` is a command that talks to the running ym over a Unix socket. Every agent ym starts can
run it, because it inherits `YM_SOCKET`, so sessions start, brief and stop each other through it.
You can use it too, from any terminal:

```sh
ym ctl list                                # every worktree, its agents and their status
ym ctl summary myrepo/fix-login            # commits, diffstat, queued messages, last agent message
ym ctl spawn myrepo fix-login --prompt "Fix the redirect loop after login"
ym ctl send myrepo/fix-login "Add a test for the expired-token case"
ym ctl interrupt myrepo/fix-login          # press Escape in that agent
ym ctl stop myrepo/fix-login               # close its sessions, keep the worktree
ym ctl open myrepo/fix-login               # start them again, resuming its agents
ym ctl reopen myrepo/fix-login             # stop and open: restart its agents on the same conversations
ym ctl remove myrepo/fix-login             # asks you to confirm in the deck
ym ctl stage "step 2"                      # from an agent: say what this session is at
ym ctl stage "in review" --worktree myrepo/fix-login  # the same for a worktree's active session
```

- Messages sent with `send` are queued and delivered when the worktree's agent is idle, so they
  never land in the middle of a turn or on a permission prompt. A `send` to a worktree that is not
  open opens it in the background with the message as the first prompt.
- They also wait while you are typing in that agent, so they never land in your own half-written
  message. ym can't read the input box, so it goes by your keys: any key marks a draft, and the
  mark goes when the agent starts the turn after your Enter, or on ctrl+c or Esc twice. The
  agent pane's top border shows `1 queued · waiting for your draft` in yellow, and `ym ctl list`
  shows `held`. A draft you abandon keeps the queue waiting until you clear it.
- `spawn` creates the worktree and starts an agent on the prompt without switching your view.
  The reply says the agent's name (`… started a claude agent named Squirtle`), picked from
  `agent_names` (see [config.md](config.md)).
- `remove` never deletes on its own: it raises ym's usual remove panel the next time you are on
  the deck.
- `ym ctl` cannot answer an agent's permission prompt or question. The `!` badge and the attention
  key show which worktree is waiting; you answer there.
- A repo's main checkout is `<repo>/main` (`ym ctl open <repo>/main`). `ym ctl` refuses to spawn
  or remove it.
- `stage` sets a short word for what a session is at (`plan`, `step 2`, `review`), at most 16
  characters, shown in the session list's box and in `ym ctl list`. Run from an agent, it is for
  that agent's own session: ym gives each agent it starts a `YM_AGENT` token. With `--worktree`,
  it is for that worktree's active session. `--clear` removes it. ym keeps it with the session, so it
  is back after a restart.

Each ym instance has its own socket under `~/.local/state/ym/` (or a short per-user directory under
`/tmp` when that path is too long for a socket). Programs started by ym get its path in
`YM_SOCKET`; from elsewhere, `ym ctl` finds the one running instance and asks you to set
`YM_SOCKET` when there are several.

## Feature managers

A feature split into several sessions can have its own manager: an ordinary worktree session that
plans the feature, starts the other sessions with `ym ctl` and tracks what blocks what. A manager
needs no special support. Start one with a skill as its first prompt:

```sh
ym ctl spawn <repo> <name> --prompt "/feature-coordinator <ticket>"
```

ym can also record the feature, so the manager and anyone else can see which worktrees belong to
it. A feature lives in one repo and has one manager; a worktree belongs to one feature at most.

```sh
ym ctl feature new myrepo abc-100 --manager myrepo/abc-100-coordinator --notes ~/notes/abc-100
ym ctl spawn myrepo abc-101-api --base abc-100 --feature abc-100 --prompt -   # joins the feature
ym ctl feature add myrepo/abc-100 myrepo/some-older-worktree
ym ctl feature list                   # manager, worktrees and gh stacks of every feature
ym ctl list --feature myrepo/abc-100
ym ctl feature drop myrepo/abc-100    # forget it; the worktrees stay
```

`ym ctl list` marks each member with `feature=abc-100`, and the manager with
`feature=abc-100 manager`. A removed worktree drops out of its feature's list. Features are kept in
ym's state file, on this machine only.

To stack a new session's branch with `gh stack`, spawn it on the top branch, then run
`gh stack add <branch>` from the top branch's worktree. `gh stack` refuses from any other worktree.
