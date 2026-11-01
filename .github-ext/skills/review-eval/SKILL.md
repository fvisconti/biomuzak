---
name: review-eval
description: "Run the Code Reviewer over the eval cases in evals/review/cases and score recall. Use after any change to the reviewer agent, REVIEW.md, or the review instructions to decide whether the change is kept."
disable-model-invocation: true
---
Run the reviewer eval set and score it.

1. **List the cases.** Read every `evals/review/cases/*.md` file. Each case
   has: a synthetic diff (or a pointer to a real commit range), the planted
   defect, and where it lives.
2. **Run the reviewer.** For each case, invoke the **Code Reviewer** agent
   (`.github/agents/code-reviewer.agent.md`) as a subagent on the case's diff
   only — do not give it the answer key.
3. **Score.** For each case:
   - `HIT` — the reviewer reported the planted defect (same file, same root
     cause; wording may differ).
   - `PARTIAL` — the reviewer touched the area but missed the root cause.
   - `MISS` — the reviewer did not report it.
   - Count false positives: findings on the case diff that are not the
     planted defect and are not real.
4. **Report.**

   ```
   Review eval: <date>
   Cases: N | HIT: x | PARTIAL: y | MISS: z | false positives: f
   Recall: (x + 0.5y) / N
   ```

   List each MISS with the case name and a one-line guess at why the reviewer
   missed it (missing invariant in `REVIEW.md`? missing instruction? model
   weakness?).
5. **Decision rule.** A change to the reviewer (agent body, `REVIEW.md`,
   instructions, model) is **kept only if recall went up** and false
   positives did not go up. Otherwise revert the change.

The eval set grows with every incident and revert: when a real defect ships,
add a case for it. See `evals/review/README.md`.
