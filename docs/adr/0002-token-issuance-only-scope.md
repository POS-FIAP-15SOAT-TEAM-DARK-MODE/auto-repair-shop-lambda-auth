# ADR 0002 — Lambda scope: token issuance only, with a minimal DB lookup to keep a real user_id

- Status: Accepted
- Date: 2026-08-22
- Deciders: Giusier F.
- Tags: lambda, authentication, scope, security
- Related: Fase 3 issue #4

## Context

The Tech Challenge brief's original wording for the auth lambda asked for three things: (1) validate the customer's CPF, (2) look up the customer's existence and status in the database, (3) generate and return a JWT. During the Fase 3 kickoff live session, the professor (Douglas Martins) confirmed that implementing only **one** of the three — specifically token issuance — is sufficient for grading purposes.

Taking that at face value (skip CPF validation and skip a DB lookup entirely) would mean the lambda just wraps whatever `user_id` it's handed into a signed JWT with no verification at all. But the main app's JWT claims (`user_id`, `roles`) are not decorative — `PUT /service-order/:id/accept` and `/reject` (the two customer-facing service-order actions) re-parse the JWT and use `claims.UserId` to look up `customer WHERE user_id = $1`, then compare that customer against the service order's actual owner. If the lambda's `user_id` isn't a real row in that table (e.g. a hash derived from the CPF, or the CPF itself), those two endpoints silently stop working for every customer who authenticates via this lambda — the ownership lookup never finds a match.

## Decision

The lambda performs the reduced, sanctioned scope (**token issuance only** — no CPF format/checksum validation, no customer "status" check, and note there is no `status` column on `customer` in the schema at all today, so that check couldn't be implemented as literally specified regardless), **but keeps one minimal database read**: `SELECT c.user_id FROM customer WHERE c.cpf = $1`, followed by a roles lookup for that `user_id`. This is intentionally *not* the full validate-existence-and-status flow from the original brief — it exists solely so the issued token's `user_id` is a real row, keeping `accept`/`reject` functional.

This does mean the lambda needs VPC placement and RDS network access (see `auto-repair-shop-infra-db` ADR 0002 for how that's wired without a circular Terraform dependency) — a real infrastructure cost that a zero-DB-access "just wrap the input" implementation would have avoided.

## Consequences

### Positive

- `accept`/`reject` continue working correctly for any customer who logs in via this lambda, not just customers who happen to also have a staff-issued token.
- Satisfies the professor-confirmed, graded scope (token issuance) without silently breaking existing, unrelated app functionality as a side effect.
- The one DB read doubles as a free "does this CPF exist" check, even though that's not the point — a nonexistent CPF returns 404 rather than issuing a token for a `user_id` that matches nothing.

### Negative

- Requires VPC networking (private subnets, a dedicated security group, RDS ingress) that a zero-DB-access version wouldn't need — more moving infrastructure pieces, more failure surface (this exact tradeoff caused two real deploy issues: a non-ASCII security group description, and later an apostrophe in the same field — see `infra-db` ADR 0002's Notes).
- Still does not implement CPF format/checksum validation or a customer status check, so it's a deliberate partial implementation of the brief's original three-part request, not a full one. Acceptable only because the grading scope was explicitly narrowed; would need revisiting if that guidance changes.
- No password, no CPF checksum validation, and no rate limiting *inside the lambda itself* means anyone who knows or guesses a valid CPF already in the `customer` table can obtain a valid `CUSTOMER`-role token for that customer. Rate limiting on this route is handled at the API Gateway layer instead (see `auto-repair-shop-infra-k8s` ADR 0002) — this lambda has no rate limiting of its own if invoked directly via its Function URL, which is intentionally still exposed for direct testing ahead of the Gateway.

## Alternatives considered

- **A. Full three-part flow (CPF validation + status check + token).** The literal original brief. Rejected per the professor-confirmed reduced grading scope, and partly infeasible as specified regardless — there is no `status` column on `customer` in the actual schema, so "check existence and status" can only mean "check existence" today without a schema change nobody asked for.
- **B. Zero DB access — wrap the input CPF (or a hash of it) directly into the JWT's `user_id`.** The most literal reading of "token issuance only." Rejected once it became clear this breaks `accept`/`reject` for any customer authenticating this way — a real functional regression for the sake of a scope interpretation nobody explicitly asked for.
- **C. Full CPF checksum validation, but still skip the status check (since there's no such column).** A middle ground considered but not chosen — validating format on a value that's about to be looked up in the database anyway (a lookup that already fails cleanly on garbage input) doesn't add meaningful correctness for this project's scope, and it's more code to maintain a second copy of (the app's own `validateCPF` in `internal/customer/domain/customer.go` isn't cheaply reusable across repos — see the Notes below).

## Notes

- The app's own CPF-validation function (`internal/customer/domain/customer.go`, `validateCPF`) is unexported and entangled with that package's internal structures, so it isn't import-shareable across repos as-is. If checksum validation is added here later (Alternative C, or a future full-scope implementation), the ~35-line algorithm would need to be ported into this lambda's own module rather than imported directly — small and stable enough that duplication is a reasonable tradeoff versus extracting a shared Go module across 4 independently-deployed repos.
