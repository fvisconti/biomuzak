---
name: security-review
description: "Run the blocking security review: a fresh-context Security Reviewer pass on the diff. An open CRITICAL finding blocks the commit until fixed or dismissed with a written reason."
disable-model-invocation: true
argument-hint: "Optional: base ref to diff against (default: develop)"
---
Run the security review for the current change set.

1. **Identify the change set.** Base ref: $1 (default `develop`). Compute
   `git diff <base>...HEAD --stat` and the full diff.
2. **Audit.** Invoke the **Security Reviewer** agent
   (`.github/agents/security-reviewer.agent.md`) as a subagent on this change
   set. Pass it the base ref and the diff range, nothing else.
3. **Record the marker.** Append one line to
   `.review/last-reviewed-<short-sha-of-HEAD>`:
   `<ISO-8601 timestamp> security-review <finding-count> findings`.
4. **Verdict.**
   - "No findings. Verdict: clear." → the security gate is open.
   - Any open 🔴 CRITICAL → the change **must not be committed**. List each
     finding; the author must fix it or dismiss it with a written reason
     recorded in the commit message or the review log.

Do not fix findings yourself in this prompt — disposition is a human step.
