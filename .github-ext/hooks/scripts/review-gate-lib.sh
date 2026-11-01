#!/usr/bin/env bash
# review-gate-lib.sh — the commit-gate decision, shared by the two callers that
# need it:
#
#   * .github/hooks/scripts/review-gate.sh — the agent-facing hook, which can only
#     *advise* a deny (the agent reads the reason and can act on it);
#   * .githooks/pre-commit — the git floor, which *stops* the commit for a human
#     typing in a terminal, where no agent hook exists.
#
# One implementation, two verdicts: a gate that behaves differently depending on who
# ran `git commit` is a gate with a hole, and the hole is always the human path.
#
# Sourced, not executed. Callers set REVIEW_DIR first if they want a non-default one.

REVIEW_DIR="${REVIEW_DIR:-.review}"
VERIFY_CACHE="$REVIEW_DIR/verify-ok"
VERIFY_TTL="${VERIFY_TTL:-600}"

# gate_head_sha prints the short SHA of HEAD, or `unknown`.
gate_head_sha() {
	git rev-parse --short HEAD 2>/dev/null || echo unknown
}

# gate_marker_path prints the marker path for the current HEAD.
gate_marker_path() {
	printf '%s/last-reviewed-%s' "$REVIEW_DIR" "$(gate_head_sha)"
}

# gate_check_marker fails when the current HEAD has no review marker.
#
# The marker is written by /pre-pr-review, /security-review and /performance-review.
# It is keyed by SHA rather than by branch because a branch moves: a marker on the
# branch would certify a commit nobody reviewed as soon as one more commit lands.
gate_check_marker() {
	local marker sha
	marker="$(gate_marker_path)"
	sha="$(gate_head_sha)"
	if [ ! -f "$marker" ]; then
		echo "review gate open: no review marker for HEAD $sha. Run /pre-pr-review, /security-review and /performance-review on the diff, disposition every finding, then commit."
		return 1
	fi
	return 0
}

# gate_check_verify fails when `make verify` is red.
#
# The result is cached for VERIFY_TTL seconds so an agent that retries a commit after
# fixing a hook denial does not re-run the whole suite each time. The cache stores a
# timestamp, not a boolean: a stale timestamp is simply a cache miss.
gate_check_verify() {
	local now then_ts out status
	now="$(date +%s)"

	if [ -f "$VERIFY_CACHE" ]; then
		then_ts="$(cat "$VERIFY_CACHE" 2>/dev/null || echo 0)"
		case "$then_ts" in
			''|*[!0-9]*) then_ts=0 ;;
		esac
		if [ $((now - then_ts)) -lt "$VERIFY_TTL" ]; then
			return 0
		fi
	fi

	out="$(make verify 2>&1)"
	status=$?
	if [ "$status" -ne 0 ]; then
		echo "make verify is red (exit $status). Last lines: $(printf '%s\n' "$out" | tail -5 | tr '\n' ' ') — fix it before committing."
		return 1
	fi

	mkdir -p "$REVIEW_DIR"
	echo "$now" > "$VERIFY_CACHE"
	return 0
}

# gate_json_escape escapes a string for embedding in a JSON string literal.
#
# Backslashes are escaped first: doing it after the quote pass would double-escape
# the quotes it had just inserted.
gate_json_escape() {
	printf '%s' "$1" | sed -e 's/\\/\\\\/g' -e 's/"/\\"/g' | tr '\n' ' '
}
