---
description: 'Zero-drift documentation sync rules: which doc to update for which code change'
applyTo: '**/*.md,**/*.go,**/*.js,**/*.jsx,**/*.py,**/*.sql,**/docker-compose*.yml,**/Dockerfile'
---

# Documentation Instructions

Code and documentation must never drift. Before declaring a task complete, update every
document whose trigger below matches the change.

| Document | Update when |
| :--- | :--- |
| `docs/CHANGELOG.md` | **any** behaviour change a user or operator can observe: new or changed endpoint, new UI control, new migration, new config knob, changed default, bug fix, or security fix |
| `docs/DOCKER.md` | any deployment, docker-compose, Dockerfile, MinIO, or environment-variable change |
| `docs/USER_GUIDE.md` | any user-facing feature, control, or workflow change |
| `docs/CONTRIBUTING.md` | any change to the development process, review gate, or contribution rules |
| `README.md` | setup steps, architecture summary, environment variable list, or test invocation change |
| `.env.example` | any environment variable is added, renamed, re-defaulted or removed |

## Rules

- **`docs/CHANGELOG.md` is mandatory on every task.** Add one bullet under
  `## [Unreleased]` in the matching category (`Added`/`Changed`/`Deprecated`/`Removed`/
  `Fixed`/`Security`), written from the operator's point of view — what changed and what
  they can now do. No commit hashes, no file lists, no internal-only refactors. Breaking
  changes start with **BREAKING**.
- **Keep a Changelog format.** `## [Unreleased]` on top, newest release below, categories
  in the canonical order. A released section is immutable: never renumber or rewrite it.
- **Pointers are repo-relative** (`docs/DOCKER.md`). Never absolute paths.
- **Never cite a document that does not exist.** If a document is still to be written,
  write it in the same commit that starts citing it.
- **No archive directory.** A superseded document is deleted or rewritten in place; a
  second copy that has stopped being true is drift, not history. `docs/CHANGELOG.md` and
  git carry the history.
- **No mega-doc.** Agent policy lives in `.github/copilot-instructions.md`, operator
  procedure in `docs/DOCKER.md`, user-facing behaviour in `docs/USER_GUIDE.md`. Do not
  recreate one document that duplicates all of them.
- `.github/copilot-instructions.md` stays **≤ 120 lines** and tree-free; detail belongs
  in the docs above. State only what is true today: never describe a guard as checking
  something it does not.
- Prefer editing an existing section over appending; stale duplicate sections are drift
  too.
