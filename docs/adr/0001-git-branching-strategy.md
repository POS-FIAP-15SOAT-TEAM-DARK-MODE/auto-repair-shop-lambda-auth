# ADR 0001 — Git branching strategy: feature → develop → main

- Status: Accepted
- Date: 2026-08-22
- Deciders: Giusier F.
- Tags: workflow, ci-cd, governance

## Context

The Tech Challenge Fase 3 brief mandates, for all 4 repositories: a protected `main`/`master` branch with no direct commits, and mandatory Pull Requests for merges. This repo's `main` had zero commits when this branch protection was first configured, since the repo was created fresh mid-session — the first two commits (initial scaffold, Go-conventions adoption) landed directly on `main` before protection existed, because there was no branch to protect yet and no reviewable history to gate.

Like `auto-repair-shop-infra-db` and `auto-repair-shop-infra-k8s`, this repo's Terraform deploy workflow selects `stg`/`prd` via a `workflow_dispatch` input, not by git branch.

## Decision

Adopt the same two-stage flow as the other 3 Tech Challenge repos: feature/fix branches → PR into `develop`; `develop` → PR into `main`. Both branches are GitHub branch-protected: PR required to merge, enforced even for repo admins, no force-push, no branch deletion, 0 required approvals. All changes from this point forward go through this flow — the two direct-to-`main` commits that predate it are a one-time bootstrap exception, not a precedent.

## Consequences

### Positive

- Matches the brief's explicit requirement, from this point forward.
- Consistent convention across all 4 Tech Challenge repos.

### Negative

- The repo's git history has two commits that bypassed this rule entirely, since they happened before the rule existed. Documented here rather than hidden, so a reviewer isn't confused by the discrepancy.

## Alternatives considered

- **A. Trunk-based (feature → main directly).** Simpler, but inconsistent with the other repos.

## Notes

- See `auto-repair-shop`'s ADR 0003 for the fuller rationale.
