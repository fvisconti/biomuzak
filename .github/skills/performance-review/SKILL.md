---
name: performance-review
description: "Run the performance review gate: a fresh-context Performance Reviewer pass on the diff, then record the marker. Use after the code review and before the commit."
disable-model-invocation: true
argument-hint: "Optional: base ref to diff against (default: main)"
---
Run the performance review for the current change set. This is the third of the
three advisory passes, in this order: `/pre-pr-review`, `/security-review`,
`/performance-review`.

1. **Identify the change set.** Base ref: $1 (default `main`). Compute
   `git diff <base>...HEAD --stat` and the full diff. If the working tree is
   dirty, say so and review `HEAD` plus the unstaged diff.
2. **Review.** Invoke the **Performance Reviewer** agent
   (`.github/agents/performance-reviewer.agent.md`) as a subagent on this change
   set. It reads `REVIEW.md` itself; pass it the base ref and the diff range,
   nothing else. Do not pre-judge the diff or tell it what to find.
3. **Record the marker.** Append one line to `.review/last-reviewed-<short-sha-of-HEAD>`:
   `<ISO-8601 timestamp> performance-review <finding-count> findings`. Append,
   do not overwrite: the code and security passes write the same file, and each
   line is evidence that its pass ran on this exact SHA.
4. **Report for disposition.** Print the findings, then list each as an open
   item. For every finding the author must either fix it, dismiss it with a
   one-line reason, or downgrade it to a recorded follow-up. An unanswered
   finding means the change is not done.
5. **Say what a dismissal costs.** A 🟡 IMPORTANT finding dismissed as "the loop
   is small" is a decision to hold a growth limit; write the limit down so the
   next reader knows when it stops being true.

Do not fix findings yourself in this prompt. Do not re-run the code or security
reviewers: on the third pass you are evaluating cost, and a second opinion on
correctness here only re-litigates findings that are already dispositioned.
