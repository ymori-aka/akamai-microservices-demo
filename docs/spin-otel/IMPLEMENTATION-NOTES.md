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

### 2026-05-20 — `[discovery]` recommendation & product-intro still use direct LLM IP

While reading the 3 functions, found that only `shopping-assistant-service`
was migrated to the Zuplo gateway. `recommendation-service` and
`product-intro-service` still call `http://172.238.48.187:8000` directly.

Implication: after we firewalled the LLM VM to Cloudflare-only +
Caddy/HTTPS, those two functions' LLM calls **likely fail now** and fall
back to their non-LLM fallback paths. This is a pre-existing bug, NOT
caused by OTel work.

Decision: out of scope for this OTel task. OTel will actually *reveal*
this — `spin_llm_errors_total` should light up for those two functions.
Flagging here so it gets fixed separately. (Recorded; not fixing now.)

### 2026-05-20 — `[gap]` SPEC didn't specify how the helper is shared across 3 packages

Each Spin function is its own npm package with its own bundle. SPEC §4.1
said "copied into each function" but didn't pin the mechanism.

Decision: physically copy `otel.ts` into each function's `src/`
(3 identical files) rather than a shared package or symlink. Reasons:
- Spin's esbuild bundle resolves from each package's own `src/`
- A shared workspace package would need npm workspace plumbing none of
  these functions currently have
- Symlinks are fragile across the CI checkout + `spin aka deploy`
Cost: 3 copies to keep in sync. Acceptable — the file is small and stable.

### 2026-05-20 — `[deviation]` histogram sent without explicit buckets

SPEC §3.3 specified histogram buckets `5,10,25,...,10000` ms. In the
hand-rolled OTLP JSON, properly bucketing each observation client-side
is fiddly. Instead each observation is sent as an OTLP histogram data
point with `count=1, sum=value, explicitBounds=[], bucketCounts=[]`.

Effect: the collector/Prometheus gets accurate **count and sum** (so
`rate(..._sum) / rate(..._count)` = avg latency works), but loses true
bucket distribution, so `histogram_quantile()` (p95 etc.) will NOT be
meaningful for these.

Why accepted: avg latency + request counts cover 90% of the demo value.
If we later want real percentiles, the clean fix is to let the OTel
**Collector** do the bucketing — but that needs a different metric type
or a processor. Deferred.

### 2026-05-20 — `[gap]` added a third placeholder OTEL_SERVICE_VERSION

SPEC §4.4 listed only two placeholders (endpoint + token). While
writing the resource attributes (SPEC §3.1 wants `service.version` =
git short SHA), I needed a third placeholder `__OTEL_SERVICE_VERSION__`.

Decision: added it, defaulting to `"dev"` when unsubstituted. CI will
sed-replace it with `${GITHUB_SHA::7}`. Updating SPEC §4.4 mentally to
"three placeholders". Low risk — purely additive.

### 2026-05-20 — `[deviation]` collector IP hardcoded in spin.toml

SPEC §4.5 showed `allowed_outbound_hosts` containing the literal
`http://172.233.70.69:4319`. The endpoint URL itself is injected into
the TS via the `OTEL_PUBLIC_ENDPOINT` secret, but `spin.toml` is NOT
run through sed — Spin reads it as-is at deploy time.

Decision: hardcode the NodeBalancer IP in all 3 `spin.toml` files.
Risk: if the LoadBalancer IP ever changes (NB recreation), outbound
calls get blocked by Spin's allowlist and telemetry silently stops
(caught by the swallow-and-log path — no request impact).

Mitigation noted: the IP is also in `OTEL_PUBLIC_ENDPOINT` secret, so a
future improvement is to template spin.toml in CI too. Deferred — IP is
stable as long as the NB isn't deleted.

### 2026-05-20 — `[gap]` LLM child-span parent linking via lastSpanId()

SPEC §3.2 said the LLM call is a "child span" but didn't specify the
mechanism. The hand-rolled tracer tracks spans in an array; I added
`tracer.lastSpanId()` and pass it as `parentSpanId` when starting the
LLM span. This works because the server span is created (and pushed)
before the LLM span starts within the same request — single-threaded
JS guarantees ordering. Acceptable for this simple 2-level hierarchy.

### 2026-05-20 — `[discovery]` all 3 functions build to WASM cleanly

Verified locally before committing: `npm run build` (esbuild + j2w)
succeeds for all three with the otel.ts import. Bundle sizes ~12MB
(shopping-assistant). No WASM-incompatible APIs hit — `AbortSignal.timeout`,
`fetch`, `Math.random`, `Date.now` all available in the Spin JS runtime.

### 2026-05-20 — `[gap]` build artifacts not committed

`dist/*.wasm`, `package-lock.json`, `.spin-aka/` left untracked (CI
rebuilds from source). Only `src/*.ts` + `spin.toml` committed, matching
the pattern from the earlier Zuplo commit.

<!-- New entries appended below as implementation proceeds -->
