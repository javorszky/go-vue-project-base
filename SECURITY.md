# Security Policy

## Reporting a vulnerability

Please do **not** open a public issue for security problems.

Use GitHub's private vulnerability reporting instead: **Security → Report a vulnerability** on this repository. Reports are acknowledged within a few days.

If private reporting is unavailable, contact the repository owner directly through their GitHub profile.

## Scope

This is a project template. Vulnerabilities in the template's own code, configuration, or CI workflows are in scope. Vulnerabilities in projects *built from* the template should be reported to those projects.

## Supported versions

Only the latest commit on `main` is supported — the template has no release branches.

## Automated scanning

The repository runs Trivy, govulncheck, CodeQL, dependency review, and zizmor on every push and PR, plus a weekly scheduled scan; dependencies are kept current by Renovate. See `.github/workflows/security.yml`.
