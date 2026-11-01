---
description: 'MKGS-specific code review instructions for the Magentia Knowledge Graph Service'
applyTo: '**'
excludeAgent: ["coding-agent"]
---

# MKGS Code Review Instructions

Comprehensive code review guidelines for the Magentia Knowledge Graph Service (MKGS), a domain-blind knowledge ingestion and retrieval engine built in Go.

## Review Language

When performing a code review, respond in **English**.

## Review Priorities

When performing a code review, prioritize issues in the following order:

### 🔴 CRITICAL (Block merge)
- **Security**: Vulnerabilities, exposed secrets, authentication/authorization issues
- **Correctness**: Logic errors, data corruption risks, race conditions
- **Breaking Changes**: API contract changes without versioning
- **Data Loss**: Risk of data loss or corruption
- **MKGS Prime Directives Violated**: Domain vocabulary in code, identity/tenancy mixed with ownership, secrets in defaults, missing provenance, spec drift

### 🟡 IMPORTANT (Requires discussion)
- **Code Quality**: Severe violations of SOLID principles, excessive duplication
- **Test Coverage**: Missing tests for critical paths or new functionality
- **Performance**: Obvious performance bottlenecks (N+1 queries, memory leaks)
- **Architecture**: Significant deviations from established patterns
- **Documentation Sync**: Docs not updated to match code changes

### 🟢 SUGGESTION (Non-blocking improvements)
- **Readability**: Poor naming, complex logic that could be simplified
- **Optimization**: Performance improvements without functional impact
- **Best Practices**: Minor deviations from conventions
- **Documentation**: Missing or incomplete comments/documentation

## General Review Principles

When performing a code review, follow these principles:

1. **Be specific**: Reference exact lines, files, and provide concrete examples
2. **Provide context**: Explain WHY something is an issue and the potential impact
3. **Suggest solutions**: Show corrected code when applicable, not just what's wrong
4. **Be constructive**: Focus on improving the code, not criticizing the author
5. **Recognize good practices**: Acknowledge well-written code and smart solutions
6. **Be pragmatic**: Not every suggestion needs immediate implementation
7. **Group related comments**: Avoid multiple comments about the same topic

## Code Quality Standards

When performing a code review, check for:

### Clean Code
- Descriptive and meaningful names for variables, functions, and classes
- Single Responsibility Principle: each function/class does one thing well
- DRY (Don't Repeat Yourself): no code duplication
- Functions should be small and focused (ideally < 20-30 lines)
- Avoid deeply nested code (max 3-4 levels)
- Avoid magic numbers and strings (use constants)
- Code should be self-documenting; comments only when necessary

### Error Handling
- Proper error handling at appropriate levels
- Meaningful error messages
- No silent failures or ignored exceptions
- Fail fast: validate inputs early
- Use appropriate error types/exceptions
- Go style: explicit `if err != nil` checks

### Examples (Go)
```go
// ❌ BAD: Ignoring errors
func getUserData(userID string) map[string]string {
    result, _ := db.Query("SELECT * FROM users WHERE id = ?", userID)
    return result
}

// ✅ GOOD: Explicit error handling
func getUserData(userID string) (map[string]string, error) {
    if userID == "" {
        return nil, fmt.Errorf("user ID required")
    }
    result, err := db.Query("SELECT * FROM users WHERE id = ?", userID)
    if err != nil {
        return nil, fmt.Errorf("query user data: %w", err)
    }
    return result, nil
}
```

### Testing
- Unit tests for all business logic
- Table-driven tests for multiple scenarios (Go idiom)
- Test fixtures in `testdata/`
- Red/green TDD: behavior change requires failing test first
- Integration tests for database operations
- No test code in production paths

### Documentation
- [ ] Public APIs are documented
- [ ] Complex logic has explanatory comments
- [ ] README is updated if needed
- [ ] Breaking changes are documented

## MKGS-Specific Review Checklist

When performing a code review for MKGS changes, verify the following:

### Prime Directives (Non-negotiable)
- [ ] **Domain-Blindness** (Directive 1): No entity type, relation type, country, regulation or taxonomy literal appears in Go code. Vocabulary must be data (config, prompts, not switch statements).
- [ ] **Tenancy & Identity** (Directive 2): Tenant/user identity only from verified JWT claims; never from body fields, path segments, or query parameters. RLS policies read `app.tenant_id`/`app.user_id` only, never `scope_id`. `owner_user_id` is a bare `uuid` with no FK to a directory. Missing GUCs fail closed.
- [ ] **Magic Literals Absent** (Directive 3): Every tunable value is an environment knob with documented default and declared class (`boot` or `live`). No hardcoded values for timeouts, limits, or thresholds.
- [ ] **Provenance Mandatory** (Directive 4): Every node and edge cites at least one source chunk. Answers disclose gaps instead of speculating. Attestations are correct and complete.
- [ ] **Contract-First** (Directive 5): New/changed routes have `openapi.json` updates in same change. New knobs documented in `docs/CONFIGURATION.md` with `boot`|`live` class. All user-visible changes have `CHANGELOG.md` bullets under `## [Unreleased]`.

### Architecture & Patterns
- [ ] **Vector Search Bounds**: Only bounded k-NN through `internal/pg` helper. No distance comparison in `WHERE` clause. Similarity filters are index-side only and untrusted radius is clamped in Go.
- [ ] **Embedding Width**: Embeddings requested at `EMBEDDING_NATIVE_DIM`, truncated to `EMBEDDING_DIM` only when `EMBEDDING_MRL=true`. Query and document vectors always share width.
- [ ] **Background Work**: All background work goes through `task.Go()`. Raw `go func()` outside `internal/task` is a defect.
- [ ] **SQL Repositories**: Hand-written parameterized SQL through pgx. No ORM. Parameters properly bound.
- [ ] **HTTP Routes**: Routes registered as `"METHOD /path/{wildcard}"` on stdlib mux. Handlers use `respond`/`respondError`, never build error bodies by hand. Only `internal/httpapi` and `internal/admincli` as surfaces; they never import each other.

### Go Style
- [ ] **Follows `.github/instructions/go.instructions.md`**: Idiomatic Go, consistent with repository patterns
- [ ] **Package Scope**: Files focused; no file grows past ~800 lines
- [ ] **Red/Green TDD**: Behavior change starts with failing test. No behavior changes without red test first.
- [ ] **Error Wrapping**: Uses `fmt.Errorf(...%w, err)` for proper error chains
- [ ] **No ORM or Router**: Direct pgx usage, stdlib `net/http.ServeMux` only

### Database & SQL
- [ ] **Follows `.github/instructions/sql.instructions.md`**: PostgreSQL 18 patterns, vchord/vchord_bm25 usage
- [ ] **Migrations**: Reversible migrations, proper up/down pairs in `migrations/`
- [ ] **RLS Policies**: FORCE ROW LEVEL SECURITY, proper `app.tenant_id`/`app.user_id` checks
- [ ] **No Hardcoded Limits**: Vector search limit, pagination, batch sizes are from environment knobs

### Testing
- [ ] **Follows `.github/instructions/tests.instructions.md`**: Red/green TDD, table-driven tests, fixture policy
- [ ] **Integration Tests**: Live PostgreSQL tests in `*_integration_test.go` when needed
- [ ] **Testdata**: Fixtures stored in `testdata/`, never inline
- [ ] **Test Isolation**: No test pollution; tests pass independently and in any order
- [ ] **Race Detection**: Passes `make test-race` for concurrency safety

### Configuration & Secrets
- [ ] **Boot vs Live**: Knobs correctly classified; boot knobs frozen at start, live knobs overridable at runtime
- [ ] **No Secret Defaults**: Secrets fail fast when missing; never have default values
- [ ] **Live Secrets Write-Only**: Writable knobs for secrets never expose values over HTTP/CLI
- [ ] **Registry Entry**: Tunable added to `internal/config/registry.go` with documentation

### Documentation Sync
- [ ] **Docs-Sync Table**: Checked against `.github/instructions/docs.instructions.md`; all required docs updated
- [ ] **openapi.json**: Route changes sync'd to spec
- [ ] **CHANGELOG.md**: User-visible changes have bullets under `## [Unreleased]`
- [ ] **CONFIGURATION.md**: New knobs documented with boot|live class and defaults
- [ ] **OPERATIONS.md**: New admin operations documented

## MKGS-Specific Language & Framework Checks

When performing a code review, verify:

### Go Idioms & Patterns
- [ ] **Concurrency Safety**: Code reviewed for race conditions; passes `make test-race`
- [ ] **Interface Design**: Interfaces are small and focused; no bloated interfaces
- [ ] **Error Handling**: Uses explicit `if err != nil` checks; no error-ignoring patterns
- [ ] **Nil Handling**: Nil pointers properly checked; no nil dereferences
- [ ] **Goroutine Cleanup**: All goroutines have clear exit paths; no goroutine leaks
- [ ] **Channel Usage**: Channels properly closed by sender; receives check for closure

### PostgreSQL & pgx
- [ ] **Prepared Statements**: All queries use parameterized statements; no SQL injection risks
- [ ] **Connection Pooling**: Uses pgx pool, no raw connection leaks
- [ ] **Transaction Handling**: Transactions properly committed/rolled back
- [ ] **Index Usage**: New queries leverage appropriate indexes
- [ ] **EXPLAIN Plans**: Complex queries verified with EXPLAIN for performance

### Vector & Sparse Search
- [ ] **VectorChord Usage**: Follows `.github/instructions/vchord.instructions.md`
- [ ] **Dimension Consistency**: Query and document vectors same width
- [ ] **k-NN Bounds**: All searches use `LIMIT k`, no unbounded loops
- [ ] **Distance Comparisons**: Only in retrieval layer, never in WHERE clauses
- [ ] **Index Selection**: Correct vchord vs vchord_bm25 operator usage

## MKGS Build & Deployment

When performing a code review, verify:

### Build & Verification
- [ ] **Makefile Targets**: Changes honor existing `make` targets; no new raw commands introduced
- [ ] **make verify**: Change passes full verification suite (fmt, vet, lint, test, route/spec sync)
- [ ] **make test-race**: No race conditions detected
- [ ] **make test-int**: Integration tests pass with live PostgreSQL
- [ ] **Docker**: Dockerfile and docker-compose.yml updated if needed

### Configuration & Runtime
- [ ] **Environment Knobs**: All new config uses registry knobs, not hardcoded
- [ ] **Boot Failures**: Missing boot knobs cause startup failure (fast-fail)
- [ ] **Live Knobs**: Overrideable at runtime without restart
- [ ] **Observability**: Metrics exported via Prometheus; no silent failures

## MKGS Contract Rules

When performing a code review, enforce:

1. **Three Surfaces, One Contract**: Admin operation is a service function rendered by HTTP handler, admin UI, and CLI. `internal/admincli` and `internal/httpapi` never import each other. Every verb reachable from one surface must be reachable from all three.

2. **Audit Trail**: Every state change writes an audit row in the same transaction.

3. **Conventional Commits**: Commit messages follow `feat(scope):`, `fix(scope):`, `refactor(scope):`, `docs:` format.

## Checklist Template

When completing a review, mark each item:

```
- [x] Requirement met
- [ ] Requirement not met
- [n/a] Not applicable
```

## Examples of MKGS-Specific Issues to Flag

### ❌ Domain Vocabulary in Code (Directive 1 Violation)
```go
// BAD: Entity types hardcoded
switch entityType {
case "Person", "Organization", "Place":
    // ...
}
```

```go
// GOOD: Domain vocabulary from config
entityTypes, err := svc.GetEntityTypes(ctx, tenantID)
for _, et := range entityTypes {
    if et.ID == entity.TypeID {
        // ...
    }
}
```

### ❌ Tenancy Mix-up (Directive 2 Violation)
```go
// BAD: Identity from body/query
userID := r.URL.Query().Get("user_id")
```

```go
// GOOD: Identity from JWT
principal, err := auth.ExtractPrincipal(r)
if err != nil {
    return nil, fmt.Errorf("extract principal: %w", err)
}
tenantID := principal.TenantID
userID := principal.UserID
```

### ❌ Magic Literal (Directive 3 Violation)
```go
// BAD: Hardcoded limit
results, err := pg.KNNSearch(ctx, vector, 10)
```

```go
// GOOD: From knob
limit := cfg.Int("search_result_limit")
results, err := pg.KNNSearch(ctx, vector, limit)
```

### ❌ Unbounded Vector Search (Arch Violation)
```sql
-- BAD: No limit
SELECT embedding <=> $1 as distance FROM chunks ORDER BY distance LIMIT ALL
```

```sql
-- GOOD: Bounded k-NN
SELECT embedding <=> $1 as distance FROM chunks ORDER BY distance LIMIT $2
```

## Additional Resources

For more information on MKGS and code review:

- **MKGS Specification**: `docs/BUILD_PLAN.md` — complete system design
- **Configuration Guide**: `docs/CONFIGURATION.md` — all boot/live knobs
- **Operation Patterns**: `docs/OPERATIONS.md` — admin operations contract
- **VectorChord Guide**: `docs/VCHORD.md` — vector search patterns
- **Go Instructions**: `.github/instructions/go.instructions.md` — Go idioms for this repo
- **SQL Instructions**: `.github/instructions/sql.instructions.md` — PostgreSQL patterns
- **Test Instructions**: `.github/instructions/tests.instructions.md` — Red/green TDD policy
- **Docs Sync**: `.github/instructions/docs.instructions.md` — which docs to update

## Prompt Engineering Tips

When performing a code review, apply these principles:

1. **Start General, Then Get Specific**: Begin with architecture review, then implementation details
2. **Reference Prime Directives**: When flagging issues, cite which directive is violated
3. **Check Spec First**: Verify `openapi.json` and migration files exist before reviewing logic
4. **Test-Driven**: Expect failing test before behavior change; flag missing red test
5. **Verify Surfaces**: For admin operations, check all three surfaces (HTTP, UI, CLI) are consistent
6. **Audit Trail**: Confirm state changes have audit rows in same transaction

## Project Context

**Tech Stack**:
- Go 1.26.1 with stdlib `net/http.ServeMux` (no router, no ORM)
- PostgreSQL 18 with vchord/vchord_bm25 for vector/sparse search
- pgx/v5 for database access
- Minimal dependencies: `jackc/pgx/v5`, `joho/godotenv`, `go.uber.org/zap`, `prometheus/client_golang`, `sony/gobreaker/v2`, `modelcontextprotocol/go-sdk`, `golang-jwt/jwt/v5`, `spf13/cobra`

**Architecture**: Domain-blind knowledge graph engine; LLM pipeline extracts entities/relations, attests to source chunks, retrieves fused dense vectors (pgvector/vchord) + sparse search (vchord_bm25) + graph traversal with cited answers

**Key Files**:
- Entry point: `cmd/mkgs/main.go`
- Service layer: `internal/` packages
- Database: `internal/pg/` with migration runner
- Configuration: `internal/config/registry.go`
- HTTP API: `internal/httpapi/`
- Admin CLI: `internal/admincli/`
- Tests: `*_test.go`, `*_integration_test.go`
- Migrations: `migrations/`

**Build Tool**: `Makefile` with targets: `test`, `test-race`, `verify`, `lint`, `test-int`, `up`

**Testing**: Table-driven tests, fixtures in `testdata/`, red/green TDD mandatory

**Code Style**: 
- Follows `.github/instructions/go.instructions.md`
- Conventional commits
- No external router/ORM
- Explicit error handling
- Contract-first (openapi.json before handlers)
- Red/green TDD (failing test before behavior change)
