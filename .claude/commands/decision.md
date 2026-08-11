---
description: Record a design decision or discussion outcome into supermemory
---

Record the following into supermemory using `add_memory` with
`containerTag: sm_project_distributed-file-storage` (always pass the tag, or the memory
lands in the account-default space):

$ARGUMENTS

Before saving, search supermemory for related prior decisions. If this contradicts or
supersedes an earlier one, say so explicitly and make the new memory reference what it
replaces — don't leave two conflicting decisions sitting side by side.

Write the memory so it reads cold in three months with no surrounding conversation:

- **What was decided** — the actual call, stated plainly.
- **What was rejected** — the alternatives considered, and why they lost.
- **Why** — the reasoning, constraints, or tradeoff axis that drove it.
- **Where it lands** — the files, packages, or roadmap step (`todos/Phase N/N.M ...`) it
  affects.

Name things explicitly. No "this", "the above", or "as discussed". If the user's input is
too thin to produce a memory that survives on its own, ask for the missing piece — the
reasoning is usually what's missing, and it's the part worth keeping.
