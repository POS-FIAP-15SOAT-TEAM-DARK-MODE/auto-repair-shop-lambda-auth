# RFC 0001 — Authentication strategy for the customer-login lambda

- Status: Resolved (see Decision)
- Date: 2026-08-22
- Author: Giusier F.
- Related: issue #4, ADR 0002 (this repo)

## Summary

The plan calls for a serverless function handling customer authentication via CPF, reusing the app's existing JWT/RBAC scheme. This RFC covers the three questions issue #4 flags as open: **runtime**, **Postgres access pattern**, and **JWT secret reuse strategy** — plus the scope question that emerged once the team narrowed the requirement mid-project.

## Problem statement

We need a Lambda that, given a customer's CPF, returns a JWT the main app's existing auth middleware accepts without any changes on that side. Constraints:

- The app's JWT is `HS256`, claims are exactly `{user_id, roles[], iat, exp}` (`internal/pkg/auth/jwt.go` in `auto-repair-shop`) — no `customerId` claim exists despite some of that repo's own documentation (`security.md`) suggesting one should. Whatever this lambda issues must match the real code, not the aspirational doc.
- `user_id` is not just a label — `accept`/`reject` service-order endpoints look it up against `customer.user_id` for ownership checks. An issued token with a fabricated `user_id` silently breaks those two endpoints for anyone who logs in via this lambda.
- The account is an AWS Academy Learner Lab — no new IAM roles/OIDC providers, which shapes how (or whether) this lambda gets database and secret access.

## Proposal

**Runtime: Go, `provided.al2023` custom runtime, `arm64`.** Chosen specifically so the JWT-signing logic (claim shape, HS256, `golang-jwt/jwt/v5`) can be *ported* line-for-line from the app's own `internal/pkg/auth/jwt.go`, rather than reimplemented in a second language where the claim shape or signing details could drift from the source of truth over time.

**Postgres access: a minimal read, not the full validate+status flow.** `SELECT user_id FROM customer WHERE cpf = $1`, then a roles lookup — see ADR 0002 in this repo for the full reasoning (agreed-upon reduced scope vs. the `accept`/`reject` ownership-check dependency that forced keeping *some* DB access rather than none).

**JWT secret reuse: read from the same Secrets Manager entry `auto-repair-shop-infra-db` already creates** (`auto-repair-shop-<env>/app`, containing both `POSTGRES_PASSWORD` and `JWT_SECRET`), fetched at cold start via the AWS SDK rather than duplicated into a second secret or baked into a Terraform-managed environment variable (which would put the plaintext secret into Terraform state a second time, in a second repo).

## Alternatives considered

### Runtime: Node.js or Python instead of Go

- **For:** Node has the fastest Lambda cold start of the common runtimes; both are more common Lambda choices generally.
- **Against:** Either would mean hand-reimplementing HS256 signing and the exact `{user_id, roles}` claim shape in a second language — a second implementation of the one piece of logic that *must* stay byte-compatible with the app's own, with no shared type system to catch drift.
- **Verdict:** rejected — Go's ability to port the app's actual signing code directly outweighs Node's cold-start advantage for this use case.

### Postgres access: none at all (wrap the raw CPF, or a hash of it, into `user_id`)

- Covered in depth in this repo's ADR 0002 — rejected because it silently breaks `accept`/`reject` for lambda-authenticated customers.

### JWT secret: a second, lambda-specific secret in Secrets Manager

- **For:** decouples this repo from `infra-db`'s Secrets Manager entry; no cross-repo secret dependency.
- **Against:** a second `JWT_SECRET` value means tokens signed by this lambda and tokens signed by the main app would use *different* keys — the main app's middleware would reject this lambda's tokens outright, defeating the entire point of the exercise (byte-compatible tokens).
- **Verdict:** rejected outright — this isn't a viable alternative, just a reminder of why the two must share one secret regardless of implementation choice.

### JWT secret: pass as a plain Terraform-managed Lambda environment variable, instead of a Secrets Manager runtime fetch

- **For:** avoids a runtime API call and its associated IAM permission (`secretsmanager:GetSecretValue`) and failure mode.
- **Against:** puts the plaintext secret into this repo's Terraform state as well as `infra-db`'s (state files aren't inherently more exposed than Secrets Manager, but doubling the number of places a live secret's plaintext is stored is a real, if modest, increase in exposure surface for no functional benefit).
- **Verdict:** rejected in favor of the runtime fetch — the extra IAM permission needed (`secretsmanager:GetSecretValue`, scoped to one ARN) is a smaller cost than duplicating the secret's storage.

## Open questions / risks

- CPF format/checksum validation is not implemented (see ADR 0002) — anyone submitting a syntactically-invalid CPF just gets a clean 404 from the "customer not found" path rather than a more specific 400. Acceptable for the current scope; worth adding if the full three-part design is ever picked back up.
- No rate limiting inside the lambda itself — relies entirely on the API Gateway's throttle (`auto-repair-shop-infra-k8s` ADR 0002) once traffic goes through the Gateway. The lambda's own Function URL, kept for direct testing, has no such protection if someone discovers and hits it directly instead of going through the Gateway.

## Decision

Recorded as `docs/adr/0002-token-issuance-only-scope.md` in this repo (scope) and reflected in `cmd/lambda/main.go` / `internal/authtoken/`, `internal/repository/`, `internal/secrets/` (implementation).
