# Spin OTel — Implementation Notes

Running log of decisions, deviations, compromises, and unexpected findings
made *during* implementation. Read alongside `SPEC.md`.

Each entry: timestamp, what happened, why.

---

## Format

- **`[deviation]`** = spec said X, we did Y
- **`[gap]`**       = spec didn't say, we had to choose
- **`[compromise]`** = spec ideal not feasible, took shortcut
- **`[discovery]`** = learned something that changes the picture

---

## Log

### 2026-05-20 — Phase 2 implementation start

Phase 1 confirmed live before starting Phase 2:
- `otel-collector-public` LoadBalancer: `172.233.70.69:4319`
- No-auth POST → HTTP 401 ✓
- Authed POST → HTTP 200 ✓
- `OTEL_PUBLIC_ENDPOINT` and `OTEL_BEARER_TOKEN` set in GitHub Secrets

---

<!-- New entries appended below as implementation proceeds -->
