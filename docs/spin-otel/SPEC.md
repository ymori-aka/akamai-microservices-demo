# Spin OpenTelemetry Integration — Specification

Status: Phase 1 implemented and verified. Phase 2 design (pre-implementation).
Last updated: 2026-05-20

---

## 1. Goal

Emit **traces and metrics** from the three Spin functions running on
**Akamai Functions** so they show up in the same Grafana stack
(Tempo for traces, Prometheus for metrics) used by the LKE microservices.

Three functions in scope:

- `shopping-assistant-service` (chat → Zuplo → Gemma)
- `recommendation-service` (product recommendation via Gemma)
- `product-intro-service` (product intro generation via Gemma)

Out of scope for this iteration:

- Log aggregation (Akamai Functions ships its own log mechanism;
  Loki integration deferred).
- mTLS for the public collector endpoint
  (Bearer token + IP allowlist sufficient for the demo).
- Sampling tuning — every span is exported.

---

## 2. Architecture

```
┌──────────────────────────────────────────────────┐
│  Akamai Functions (Cloudflare Workers runtime)   │
│                                                  │
│  shopping-assistant ──┐                          │
│  recommendation     ──┼── POST /v1/traces        │
│  product-intro      ──┘   POST /v1/metrics       │
│                       Authorization: Bearer …    │
└──────────────────────────┬───────────────────────┘
                           │ HTTP (no TLS)
                           ▼
        ┌─────────────────────────────────────┐
        │  Linode NodeBalancer  172.233.70.69 │  ← Phase 1 (done)
        │            :4319                    │
        └────────────────┬────────────────────┘
                         ▼
        ┌─────────────────────────────────────┐
        │  otel-collector (LKE / monitoring)  │
        │  • otlp/public receiver (Bearer)    │
        │  • bearertokenauth extension        │
        │  • batch + memory_limiter           │
        │                                     │
        │  ┌────────► Tempo (traces)          │
        │  └────────► Prometheus (metrics)    │
        └─────────────────────────────────────┘
```

### 2.1 Public endpoint

- URL: `http://172.233.70.69:4319` (also reachable via
  `http://172-233-70-69.ip.linodeusercontent.com:4319`)
- Stored in GitHub Secret `OTEL_PUBLIC_ENDPOINT`
- Plain HTTP. **TLS deliberately omitted** for the demo —
  Bearer token over HTTP is acceptable because:
  - The endpoint only accepts OTLP JSON, no read paths
  - Token rotation is cheap (single GitHub Secret + workflow rerun)
  - Adding cert-manager/Caddy would add 2+ moving parts for marginal
    benefit at demo scale

### 2.2 Authentication

- Single shared bearer token (`OTEL_BEARER_TOKEN`)
- Stored in:
  - GitHub Secrets (for CI build-time injection)
  - K8s Secret `otel-bearer-token` in `monitoring` namespace
    (mounted as `OTEL_BEARER_TOKEN` env var in collector)
- Validated by OTel Collector's `bearertokenauth` extension
- Header format: `Authorization: Bearer <token>`

---

## 3. What gets emitted

### 3.1 Resource attributes (on every batch)

| Key | Value | Example |
|---|---|---|
| `service.name` | function name | `shopping-assistant-service` |
| `service.version` | git short SHA at build time | `233a40d` |
| `deployment.environment` | `akamai-functions` | static |
| `cloud.provider` | `akamai` | static |
| `cloud.platform` | `akamai_functions` | static |

### 3.2 Traces — one span per request

Each function emits exactly one span per inbound HTTP request:

| Attribute | Value |
|---|---|
| Span name | `<route> <method>` (e.g. `POST /chat`) |
| Span kind | `SERVER` |
| `http.method` | request method |
| `http.route` | matched route |
| `http.status_code` | response status |
| `error` | true on non-2xx (no PII in description) |

When the function calls an upstream LLM, a **child span** is added:

| Attribute | Value |
|---|---|
| Span name | `llm.chat.completions` |
| Span kind | `CLIENT` |
| `llm.endpoint` | URL (host part only, no path query) |
| `llm.model` | model name (e.g. `google_gemma-4-26B-A4B-it-Q4_K_M.gguf`) |
| `llm.tokens.prompt` | from response.usage if present |
| `llm.tokens.completion` | from response.usage if present |
| `llm.tokens.total` | from response.usage if present |
| `http.status_code` | upstream response status |

### 3.3 Metrics — per-function counters and a histogram

Emitted on every request via OTLP/HTTP JSON:

| Name | Type | Unit | Attributes |
|---|---|---|---|
| `spin_requests_total` | counter | `1` | `service`, `route`, `status_code` |
| `spin_request_duration_ms` | histogram | `ms` | `service`, `route`, `status_code` |
| `spin_llm_tokens_total` | counter | `1` | `service`, `model`, `kind={prompt,completion}` |
| `spin_llm_errors_total` | counter | `1` | `service`, `model`, `reason` |

Histogram buckets (ms): `5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000`.

These names use the `spin_` prefix to clearly differentiate from the
existing `boutique_` (OTel Collector namespace) and `k6_` metrics in
Prometheus. The OTel Collector's Prometheus exporter is configured with
`namespace: boutique` for backwards compat, so these will appear as
`boutique_spin_requests_total` etc. in Prometheus — Grafana queries
should account for this.

---

## 4. Implementation approach

### 4.1 Library: hand-rolled OTLP/HTTP JSON

Reason: full `@opentelemetry/sdk-trace-*` packages drag in lots of code
(stream handling, context propagation polyfills) that is non-trivial to
bundle into a Spin WASM target via `j2w`. We only need to POST JSON.

Manifest of the helper module (`src/otel.ts`, copied into each function):

```ts
export function makeTracer(serviceName: string): Tracer
  // returns:
  //   - withSpan(name, attrs, fn)  — wraps a fn in a span
  //   - recordCounter(name, value, attrs)
  //   - recordHistogram(name, value, attrs)
  //   - flush()  — POSTs pending spans+metrics to the collector
```

Behavior:
- Spans buffered in-memory during a request.
- `flush()` is called once at the end of each request handler.
- One POST to `/v1/traces` and one to `/v1/metrics` per request.
- Failures are **logged via `console.error` and swallowed** — telemetry
  must never break the request path.
- A 2-second per-POST timeout via `AbortSignal.timeout(2000)`.

### 4.2 IDs

- `traceId`: 16 random bytes, hex-encoded (32 chars).
- `spanId`: 8 random bytes, hex-encoded (16 chars).
- `Math.random()` is acceptable — no Web Crypto guarantee in Spin runtime.

### 4.3 Time

- `Date.now() * 1_000_000` for nanoseconds (OTLP expects nanos).
- Acceptable precision loss for ~1ms LLM-call durations.

### 4.4 Configuration injection

The helper reads two constants from the source file:

```ts
const OTEL_ENDPOINT = "__OTEL_ENDPOINT__";
const OTEL_BEARER_TOKEN = "__OTEL_BEARER_TOKEN__";
```

CI replaces these via `sed` **before `npm run build`** in each function's
deploy step. Same pattern already used for `__ZUPLO_API_KEY__`.

### 4.5 spin.toml changes

Each function's `allowed_outbound_hosts` must include the collector URL:

```toml
allowed_outbound_hosts = [
  "https://chat-ai-ai-gateway-34777e2.zuplo.app",
  "http://172.233.70.69:4319",
]
```

---

## 5. CI changes (`deploy.yml`)

For each of the three Spin function deploy steps:

```yaml
env:
  OTEL_ENDPOINT: ${{ secrets.OTEL_PUBLIC_ENDPOINT }}
  OTEL_BEARER_TOKEN: ${{ secrets.OTEL_BEARER_TOKEN }}
run: |
  sed -i "s|__OTEL_ENDPOINT__|${OTEL_ENDPOINT}|g" src/index.ts src/otel.ts
  sed -i "s|__OTEL_BEARER_TOKEN__|${OTEL_BEARER_TOKEN}|g" src/index.ts src/otel.ts
  npm install && npm run build
  spin aka app deploy --no-confirm
```

(Shopping-assistant already has the same pattern for `__ZUPLO_API_KEY__`.)

---

## 6. Verification

After deploy, the following commands must succeed:

```bash
# 1. Request the shopping assistant — produces 1 server + 1 client span
curl -X POST https://<shopping-assistant>/chat \
  -d '{"messages":[{"role":"user","content":"hi"}]}'

# 2. Trace must appear in Tempo (search by service.name)
# 3. Metric must show in Prometheus:
#    boutique_spin_requests_total{service="shopping-assistant-service"} >= 1
```

A new GitHub workflow `verify-spin-otel.yml` will automate steps 2–3.

---

## 7. Out-of-scope decisions (record, not implement)

- **Buffering across requests**: not implemented. Each request flushes
  its own telemetry. Risk: slightly higher overhead per request, but
  Spin/WASM cold-starts make cross-request buffering messy.
- **W3C trace context propagation**: not implemented in the outbound
  Zuplo call. If we ever want end-to-end traces all the way through
  to the LLM, we'd add `traceparent` header injection.
- **TLS on the collector**: see §2.1.
- **gRPC OTLP**: HTTP/JSON only. gRPC needs HTTP/2 which is fine on
  Cloudflare Workers but adds a protobuf dependency.
