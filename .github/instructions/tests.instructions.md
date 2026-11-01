---
description: 'Testing rules for this repository: red/green TDD, table-driven Go tests, fixture policy'
applyTo: '**/*_test.go,**/testdata/**,**/tests/**'
---

# Testing Instructions

## Golden rule — red/green TDD

Every change starts with a failing test. Write it, watch it fail for the right reason,
implement the minimum to make it pass, then refactor with the suite green. No
production code without a test. Never report a task done while any test is red.

## Layers

| Layer | Command | Requirements |
| :--- | :--- | :--- |
| Unit (Go) | `go test -race ./...` | offline, no DB, no network, no MinIO. Must stay fast. |
| Unit (Python) | `pytest` (in `audio-processor/`) | offline; `es.MonoLoader`/`es.Extractor` mocked, no real audio. |
| Unit (Frontend) | `npm test` (in `frontend/`) | offline; API client mocked, no live backend. |
| Race | `go test -race ./...` | required before every commit touching concurrency |
| Full stack | `docker compose up` | manual smoke; not part of the unit gate |

## Style

- Table-driven with subtests:
  ```go
  tests := []struct{ name string; in string; want string }{...}
  for _, tt := range tests {
      t.Run(tt.name, func(t *testing.T) { ... })
  }
  ```
- One invariant per case; the case name states the invariant.
- No live audio-processor, MinIO or Postgres calls. Use interfaces, fakes,
  `go-sqlmock` and `httptest.Server`.
- Every bug fix ships the regression test that reproduces the original bug.
- Fixtures live in `testdata/`, are small and deterministic. Never commit generated
  corpora or large audio files.
- Never assert on log output or timing; assert on returned values and DB state.

## Documentation in tests

Each test file opens with a short godoc block stating **Objective**, **Given/When/Then**
and the **Edge cases** covered. New suites are recorded in `docs/CHANGELOG.md` under
`## [Unreleased]`.
