---
name: pre-pr-review
description: "Run the pre-commit review gate: go test -race + go vet, then a fresh-context Code Reviewer pass on the diff, then record the review marker. Use before every commit of a finished change."
disable-model-invocation: true
argument-hint: "Optional: base ref to diff against (default: main)"
---
Run the pre-PR review gate for the current change set. Work through these
steps in order and do not stop at a failure — report it and continue where
possible.

1. **Verify.** Run `go test -race ./...` and `go vet ./...`. If either is red,
   stop here: list the failing packages and do not review. A review of red code
   is wasted attention.
2. **Identify the change set.** Base ref: $1 (default `main`). Compute
   `git diff <base>...HEAD --stat` and the full diff. If the working tree is
   dirty, say so and review `HEAD` plus the unstaged diff.
3. **Review.** Invoke the **Code Reviewer** agent
   (`.github/agents/code-reviewer.agent.md`) as a subagent on this change set.
   It reads `REVIEW.md` and the review instructions itself; pass it the base
   ref and the diff range, nothing else.
4. **Record the marker.** After the review completes, write the file
   `.review/last-reviewed-<short-sha-of-HEAD>` containing one line:
   `<ISO-8601 timestamp> code-review <finding-count> findings`. Create
   `.review/` if missing. This marker is advisory evidence that the pass ran on
   this exact SHA.
5. **Report for disposition.** Print the reviewer's findings, then list each
   finding as an open item. For every finding the author must either fix it
   or dismiss it with a one-line reason. An unanswered finding means the
   change is not done.

Do not fix findings yourself in this prompt — disposition is a human step.
If the author says "fix them", that is a new task.
