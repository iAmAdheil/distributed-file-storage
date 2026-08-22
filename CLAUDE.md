# Distributed File Storage

A Go P2P content-addressed store being extended into an S3-compatible distributed object
store. The build roadmap lives in the Obsidian vault at `todos/` (see
`todos/Distributed File Storage — Extension Roadmap.md` and the Build Board kanban).

## Memory: supermemory is the source of truth

Project history — discussions, decisions, tradeoffs, resolved doubts — lives in
**supermemory**, not in local memory files and not in this file. The MCP server is
configured in `.mcp.json` and authenticates via **OAuth** (the endpoint rejects API keys —
see the gotcha memory in supermemory before touching that config).

Do **not** write to the file-based memory directory under
`~/.claude/projects/.../memory/`. It was deliberately removed in favour of supermemory.

### Always pass the container tag

Every supermemory call — `search_memory`, `add_memory`, `listMemories` — must pass:

```
containerTag: sm_project_distributed-file-storage
```

Omitting it silently falls back to the account-default space (`sm_project_default`), where
a search returns zero results and a save lands in the wrong place. Do not fix this with
`set-active-tag`: the active space is account-wide and would leak into other projects.
Pass the tag explicitly, every call.

### Recall — at the start of substantive work

Before planning or building anything non-trivial, search supermemory for prior context on
the task at hand (the todo/phase being worked, the subsystem being touched, the design
question being asked). Past decisions constrain present ones; do not re-litigate a
tradeoff that was already settled, and do not contradict an earlier decision without
saying so explicitly.

If recall returns nothing relevant, proceed — don't announce the empty result.

### Save — when something is decided

Save to supermemory whenever, in the course of a session:

- **A design decision is made** — what was chosen, what was rejected, and *why*.
- **A tradeoff is settled** — the axis (consistency vs availability, simplicity vs
  throughput), the call, and the reasoning.
- **A roadmap step is completed** — which todo/phase, what was actually built, anything
  that diverged from the note in `todos/`.
- **A doubt is resolved** — via `dfs-guide`, `go-mentor`, or ordinary discussion: the
  question and the understanding that resolved it.
- **A constraint or gotcha is discovered** — something that will bite again later.

Save the *conclusion and its reasoning*, not a transcript. One decision per memory. Write
it so it makes sense read cold in three months, with no surrounding conversation: name the
files, packages, and concepts explicitly rather than saying "this" or "the above".

Do **not** save: routine code changes (git history covers those), file structure, anything
already written down in `todos/`, or context that only matters within the current session.

Saving is automatic — no need to ask permission each time. Mention briefly what was saved.

## Working style

- Present the plan before executing multi-step work; surface genuine decisions rather than
  guessing. The user is optimising for **learning and competency** — they want to
  understand and defend every design choice, not just receive working code.
- Verify references against real sources (papers, official docs) before citing them.
- Mid-coding doubts → the `dfs-guide` **skill** (`/dfs-guide`). It loads Socratic guide
  mode into this session rather than spawning a subagent, because clearing a doubt is a
  back-and-forth: it hints, hands the turn back, and escalates only if you're still stuck.
  While it's active the assistant will not write the code for the step you're on. It
  releases on an explicit "just tell me" or as soon as you move to a different task.
- Post-commit Go quality review → the `go-mentor` **subagent** (hand it a commit hash).
  Stays a subagent on purpose: it's a one-shot report over a large read-only pile (`git
  show`, `git grep`, files-at-commit), and that bulk is better kept out of this context.
- Code quality convention (`notes.md`): important things at the top of a file, basic
  helper functions at the bottom.
