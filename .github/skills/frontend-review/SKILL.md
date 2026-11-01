---
name: frontend-review
description: "Run a focused review of the React / Vite / Chakra frontend: npm run lint green, then a Code Reviewer pass scoped to frontend/ against the API-client, context-state, theming, and no-secrets rules."
disable-model-invocation: true
argument-hint: "Optional: base ref to diff against (default: main)"
---
Run a focused review of the frontend for the current change set. Work through
these steps in order and do not stop at a failure — report it and continue
where possible.

1. **Verify.** Run `npm run lint` inside `frontend/`. If it is red, stop here:
   list the lint errors and do not review. A review of red code is wasted
   attention.
2. **Identify the change set.** Base ref: $1 (default `main`). Compute
   `git diff <base>...HEAD --stat -- frontend/` and the full diff for
   `frontend/`. If the working tree is dirty, say so and review `HEAD` plus the
   unstaged diff. If the diff does not touch `frontend/`, say so and stop —
   there is nothing to review.
3. **Review.** Invoke the **Code Reviewer** agent
   (`.github/agents/code-reviewer.agent.md`) as a subagent on this change set,
   scoped to `frontend/`. It reads `REVIEW.md` and
   `.github/instructions/frontend.instructions.md` itself; pass it the base
   ref, the diff range, and the scope, nothing else.
4. **Record the marker.** After the review completes, append one line to
   `.review/last-reviewed-<short-sha-of-HEAD>`:
   `<ISO-8601 timestamp> frontend-review <finding-count> findings`. Create
   `.review/` if missing.
5. **Report for disposition.** Print the reviewer's findings, then list each
   finding as an open item. For every finding the author must either fix it
   or dismiss it with a one-line reason. An unanswered finding means the
   change is not done.

Do not fix findings yourself in this prompt — disposition is a human step.
If the author says "fix them", that is a new task.
