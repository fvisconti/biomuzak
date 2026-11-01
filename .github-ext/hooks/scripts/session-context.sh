#!/usr/bin/env bash
# session-context.sh — injects the review-gate state at the start of every session.
#
# Only `additionalContext` is consumed for `sessionStart`; a `systemMessage` field is
# ignored there, so the text is emitted under the key the event actually reads.
#
# The gate state is injected rather than left to the instructions file because the
# instructions cannot know it: whether *this* HEAD is reviewed is a fact about the
# working tree, and telling the agent up front is what makes "run /pre-pr-review" the
# obvious next step instead of a surprise at commit time.
set -uo pipefail

# shellcheck source=review-gate-lib.sh
. "$(dirname "$0")/review-gate-lib.sh"

branch="$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo unknown)"
head_sha="$(gate_head_sha)"
marker="$(gate_marker_path)"

if [ -f "$marker" ]; then
	review="reviewed — $(head -1 "$marker" 2>/dev/null)"
else
	review="not reviewed (advisory) — run /pre-pr-review, /security-review, /performance-review when you want; they do not block a commit"
fi

msg="Review gate — branch: $branch, HEAD: $head_sha. Reviews: $review. A red 'make verify' blocks a commit; the three reviews are advisory and record a marker as evidence (see REVIEW.md)."
printf '{"additionalContext":"%s"}\n' "$(gate_json_escape "$msg")"
exit 0
