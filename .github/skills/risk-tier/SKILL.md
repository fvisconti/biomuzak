---
name: risk-tier
description: "Classify the current change set into a risk tier (Low/Medium/High) and ask the author to confirm it. The tier decides how much human review the change gets before merge."
disable-model-invocation: true
argument-hint: "Optional: base ref to diff against (default: main)"
---
Classify the risk tier of the current change set.

1. **Identify the change set.** Base ref: $1 (default `main`).
2. **Classify.** Invoke the **Risk Classifier** agent
   (`.github/agents/risk-classifier.agent.md`) as a subagent. It returns one
   line: `TIER: Low|Medium|High — <reason>`.
3. **Confirm with the author.** Print the proposed tier and its reason, then
   ask the author to confirm or raise it. The author may raise a tier but the
   classifier's tier is the floor, not the ceiling. Record the confirmed tier.
4. **State the consequence.**
   - **Low** — automated review + security review green is enough; the
     author's own approval counts.
   - **Medium** — one person reads the diff and the evidence (test output,
     `go test -race ./...` result) before merge.
   - **High** — the author actually reads the diff line by line and the
     evidence before merge.

The confirmed tier goes into the PR template's `## Risk` section.
