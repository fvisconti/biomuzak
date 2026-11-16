## Testing

What was run and what it showed. Paste the key output, not a summary of a
summary: the commands (`go test -race ./...`, `go vet ./...`, `pytest`), the
failing test that went red first, and the command that settles any "needs
verification" finding from the review.

## Risk

Tier (Low / Medium / High) and why — from `/risk-tier`, confirmed by the
author. High tier requires the author to have read the diff and the evidence
before merge.

## How to revert

The exact command or migration that undoes this change
(e.g. `git revert <sha>`, or a new numbered migration in `db/migrations/` that
reverses the schema change), and what state the data is in afterwards.
