---
description: 'Zero-drift documentation sync rules: which doc to update for which code change'
applyTo: '**/*.md,**/*.go,**/*.js,**/*.sql,**/openapi.json'
---

# Documentation Instructions

Code and documentation must never drift. Before declaring a task complete, update every document
whose trigger below matches the change.

| Document | Update when |
| :--- | :--- |
| `CHANGELOG.md` | **any** behaviour change a user or operator can observe: new or changed endpoint, new UI control, new migration, new config knob, changed default, bug fix, or security fix |
| `docs/BUILD_PLAN.md` | architecture, data model, API surface, dependency set or epic scope changes. This is the specification: when implementation and plan disagree, decide which is right and change the other in the same commit |
| `docs/CONFIGURATION.md` | any environment knob is added, renamed, re-defaulted or removed |
| `docs/VCHORD.md` | any vchord/vchord_bm25/pg_tokenizer change: index option, operator class, GUC, tokenizer object or engine version. This is the reference; the rules are `.github/instructions/vchord.instructions.md` and the two are updated together |
| `docs/OPERATIONS.md` | any operator-facing procedure changes: ingestion stages, retrieval pipeline, graph maintenance, job lifecycle, backup/restore, troubleshooting, or an admin UI control |
| `openapi.json` | any route, method, path parameter, query parameter, payload field, error code or SSE event is added, changed or removed |
| `README.md` | setup steps, architecture summary, environment variable list, or test invocation change |

## Rules

- **`CHANGELOG.md` is mandatory on every task.** Add one bullet under `## [Unreleased]` in the
  matching category (`Added`/`Changed`/`Deprecated`/`Removed`/`Fixed`/`Security`), written from the
  operator's point of view — what changed and what they can now do. No commit hashes, no file lists,
  no internal-only refactors. Breaking changes start with **BREAKING**. `make docs-check` fails if
  `CHANGELOG.md` is missing or has lost its `## [Unreleased]` section.
- **Keep a Changelog format.** `## [Unreleased]` on top, newest release below, categories in the
  canonical order. A released section is immutable: never renumber or rewrite it.
- **Pointers are repo-relative** (`docs/OPERATIONS.md`). Never absolute paths — `make docs-check`
  fails the build on them.
- **Never cite a document that does not exist.** `make docs-check` fails when tracked markdown or Go
  mentions a `docs/*.md` path that is missing from the tree. If a document is still to be written,
  write it in the same commit that starts citing it — an allow-list of "planned" dead paths only
  teaches the guard to stop looking.
- **No archive directory.** A superseded document is deleted or rewritten in place; a second copy
  that has stopped being true is drift, not history. `CHANGELOG.md` and git carry the history.
- **`docs/BUILD_PLAN.md` is a specification, not a diary.** Never leave a section describing tables,
  routes or packages that do not exist. The plan is updated in the same commit that proves it wrong.
- **No mega-doc.** Agent policy lives in `.github/copilot-instructions.md`, prime directives and the
  data model in `docs/BUILD_PLAN.md` §2–5, operator procedure in `docs/OPERATIONS.md`, knobs in
  `docs/CONFIGURATION.md`. Do not recreate one document that duplicates all four.
- `.github/copilot-instructions.md` stays **≤ 120 lines** and tree-free; detail belongs in the docs
  above. State only what is true today: never describe a guard as checking something it does not.
- Bump the cache-busting `?v=` query in `static/admin/index.html` whenever you edit any
  `static/admin/*.js` or `*.css` file, so operators never run a stale admin UI.
- Prefer editing an existing section over appending; stale duplicate sections are drift too.
- Run `make docs-check` before committing documentation changes.
