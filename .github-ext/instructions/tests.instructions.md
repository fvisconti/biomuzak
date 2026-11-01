---
description: 'Testing rules for this repository: red/green TDD, table-driven Go tests, fixture policy'
applyTo: '**/*_test.go,**/testdata/**,**/benchmarks/**'
---

# Testing Instructions

## Golden rule — red/green TDD

Every change starts with a failing test. Write it, watch it fail for the right reason, implement the
minimum to make it pass, then refactor with the suite green. No production code without a test.
Never report a task done while any test is red.

## Layers

| Layer | Command | Requirements |
| :--- | :--- | :--- |
| Unit | `make test` | offline, no DB, no network, no router. Must stay fast. |
| Race | `make test-race` | required before every commit touching concurrency |
| Integration | `make test-int` | live `DATABASE_URL`; `t.Skip` when unset |
| Gate | `make verify` | fmt-check + vet + lint + test |

## Style

- Table-driven with subtests:
  ```go
  tests := []struct{ name string; in string; want string }{...}
  for _, tt := range tests {
      t.Run(tt.name, func(t *testing.T) { ... })
  }
  ```
- One invariant per case; the case name states the invariant.
- No live LLM, router or MinIO calls. Use interfaces, fakes and `httptest.Server`.
- Every bug fix ships the regression test that reproduces the original bug.
- Fixtures live in `testdata/`, are small and deterministic. Never commit generated corpora.
- Never assert on log output or timing; assert on returned values and DB state.

## Documentation in tests

Each test file opens with a short godoc block stating **Objective**, **Given/When/Then** and the
**Edge cases** covered. New suites and benchmarks are recorded in `CHANGELOG.md` and in
`docs/OPERATIONS.md` (verification section).
