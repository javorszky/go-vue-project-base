# Contributing

Thanks for considering a contribution. This document covers the mechanics; the [README](README.md) covers what the project is and how it's put together.

## Development setup

1. Install the prerequisites from the [README's Quick start](README.md#prerequisites) (Go, Node, golangci-lint, lefthook, shellcheck).
2. Clone, then activate git hooks once:
   ```bash
   lefthook install
   ```
3. Install frontend dependencies:
   ```bash
   cd frontend && npm install
   ```

Day-to-day commands live in the `Makefile` — `make test`, `make lint`, `make lint-fix`. See the README's *Development commands* section for the full list.

## Before you open a PR

- `make lint && make test` must pass. CI runs the same checks plus security scans.
- Keep the diff focused: one logical change per PR.
- Update documentation that your change makes stale — including `.ai/index.md` if you add, remove, or rename a symbol.

## Commit conventions

- [Conventional Commits](https://www.conventionalcommits.org/) subjects: `fix(server): …`, `docs(readme): …`.
- If AI assisted with a commit, record it with an `Assisted-by:` trailer rather than `Co-authored-by:` — this keeps AI involvement visible without skewing the contributor graph:
  ```
  Assisted-by: <model name> <noreply@anthropic.com>
  ```

## Code style

- Go: enforced by `.golangci.yml` (gofumpt, gci import grouping, and a strict linter set). Comments explain *why*, not *what*.
- Frontend: `<script setup lang="ts">` only; formatting via Prettier; linting via ESLint flat config with accessibility rules.
- Both are checked by the pre-commit hook, so a passing commit is a compliant one.

## Repository ownership

This template intentionally ships without a `CODEOWNERS` file — add your own in `.github/CODEOWNERS` once your team exists.
