#!/usr/bin/env bash
# review-gate.sh — the agent-facing half of the commit gate.
#
# Fires on every `preToolUse`. When the agent is about to run a `git commit`, the
# call is denied only when `make verify` is red (cached, see review-gate-lib.sh).
#
# The three reviews (/pre-pr-review, /security-review, /performance-review) are
# advisory: they record a marker as evidence but a missing marker does not deny the
# commit. A WIP commit with reviews pending is intentional and allowed.
#
# The deny is a **top-level** decision on stdout with exit 0:
#   {"permissionDecision":"deny","permissionDecisionReason":"..."}
# That is the shape the Copilot harness reads. A nested Claude-style
# `hookSpecificOutput` object is silently treated as "no decision", which is how a
# gate that looks configured stops blocking anything.
#
# Exit 0 is deliberate: a non-zero exit *also* denies, but with a generic message, so
# the agent cannot see why or how to fix it. Anything that is not a git commit prints
# nothing and exits 0, which the harness reads as "no decision".
set -uo pipefail

# shellcheck source=review-gate-lib.sh
. "$(dirname "$0")/review-gate-lib.sh"

payload="$(cat 2>/dev/null || true)"

# --- extract the command the agent wants to run ------------------------------
# The camelCase payload carries `toolArgs`; the VS Code-compatible PascalCase
# payload carries `tool_input`. Read both so the gate does not depend on which
# spelling of the event name is configured.
command=""
if command -v jq >/dev/null 2>&1; then
	command="$(printf '%s' "$payload" \
		| jq -r '.toolArgs.command // .tool_input.command // empty' 2>/dev/null || true)"
fi
if [ -z "$command" ]; then
	command="$(printf '%s' "$payload" \
		| grep -o '"command"[[:space:]]*:[[:space:]]*"[^"]*"' \
		| head -1 | sed 's/.*"command"[[:space:]]*:[[:space:]]*"//; s/"$//' || true)"
fi

# Not a commit: nothing to gate.
case "$command" in
	*git\ commit*) ;;
	*) exit 0 ;;
esac

deny() {
	printf '{"permissionDecision":"deny","permissionDecisionReason":"%s"}\n' \
		"$(gate_json_escape "$1")"
	exit 0
}

# Only the build gate can deny. The reviews are advisory and never block.
reason="$(gate_check_verify)" || deny "$reason"

# Both gates passed. Emitting `{}` is "no decision", not an approval: the normal
# permission flow still runs, so this hook never widens the agent's authority.
printf '{}\n'
exit 0
