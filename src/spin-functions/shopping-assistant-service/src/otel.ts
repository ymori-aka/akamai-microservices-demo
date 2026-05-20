// Minimal OTLP/HTTP JSON telemetry emitter for Spin functions.
//
// Hand-rolled (no @opentelemetry/* deps) so it bundles cleanly into a
// Spin WASM target via j2w. See docs/spin-otel/SPEC.md §4.
//
// CI replaces the two placeholders below at build time (sed), sourced
// from the OTEL_PUBLIC_ENDPOINT / OTEL_BEARER_TOKEN GitHub Secrets.
// If they are left as placeholders (e.g. local build), telemetry is
// silently disabled — it must NEVER break the request path.

const OTEL_ENDPOINT = "__OTEL_ENDPOINT__";
const OTEL_BEARER_TOKEN = "__OTEL_BEARER_TOKEN__";
const SERVICE_VERSION = "__OTEL_SERVICE_VERSION__"; // git short SHA, optional

const ENABLED =
  !OTEL_ENDPOINT.startsWith("__") && !OTEL_BEARER_TOKEN.startsWith("__");

// ---- ID + time helpers ----------------------------------------------------

function hex(bytes: number): string {
  let s = "";
  for (let i = 0; i < bytes; i++) {
    s += Math.floor(Math.random() * 256).toString(16).padStart(2, "0");
  }
  return s;
}
const nowNanos = () => `${Date.now()}000000`;

// ---- OTLP attribute encoding ----------------------------------------------

type Attrs = Record<string, string | number | boolean | undefined>;

function kv(attrs: Attrs) {
  const out: any[] = [];
  for (const [key, v] of Object.entries(attrs)) {
    if (v === undefined) continue;
    let value: any;
    if (typeof v === "number") {
      value = Number.isInteger(v) ? { intValue: v } : { doubleValue: v };
    } else if (typeof v === "boolean") {
      value = { boolValue: v };
    } else {
      value = { stringValue: String(v) };
    }
    out.push({ key, value });
  }
  return out;
}

// ---- Tracer ---------------------------------------------------------------

interface SpanRec {
  traceId: string;
  spanId: string;
  parentSpanId?: string;
  name: string;
  kind: number; // 2=SERVER, 3=CLIENT
  startNano: string;
  endNano: string;
  attrs: Attrs;
  error: boolean;
}

interface NumberDataPoint {
  asInt?: number;
  asDouble?: number;
  attrs: Attrs;
}

export class Tracer {
  private spans: SpanRec[] = [];
  private counters = new Map<string, NumberDataPoint[]>();
  private histos = new Map<string, { value: number; attrs: Attrs }[]>();
  readonly traceId = hex(16);
  private service: string;

  constructor(service: string) {
    this.service = service;
  }

  /** Wrap an async fn in a span. parentId optional for child spans. */
  async withSpan<T>(
    name: string,
    kind: "SERVER" | "CLIENT",
    attrs: Attrs,
    fn: (span: { setAttr: (k: string, v: any) => void }) => Promise<T>,
    parentSpanId?: string,
  ): Promise<T> {
    const rec: SpanRec = {
      traceId: this.traceId,
      spanId: hex(8),
      parentSpanId,
      name,
      kind: kind === "SERVER" ? 2 : 3,
      startNano: nowNanos(),
      endNano: "",
      attrs: { ...attrs },
      error: false,
    };
    const span = { setAttr: (k: string, v: any) => { rec.attrs[k] = v; } };
    try {
      const result = await fn(span);
      return result;
    } catch (e) {
      rec.error = true;
      rec.attrs["error"] = true;
      throw e;
    } finally {
      rec.endNano = nowNanos();
      this.spans.push(rec);
    }
  }

  /** ID of the most recently started/finished span (for parent linking). */
  lastSpanId(): string | undefined {
    return this.spans[this.spans.length - 1]?.spanId;
  }

  recordCounter(name: string, value: number, attrs: Attrs) {
    const arr = this.counters.get(name) ?? [];
    arr.push({ asInt: value, attrs });
    this.counters.set(name, arr);
  }

  recordHistogram(name: string, value: number, attrs: Attrs) {
    const arr = this.histos.get(name) ?? [];
    arr.push({ value, attrs });
    this.histos.set(name, arr);
  }

  private resource() {
    return {
      attributes: kv({
        "service.name": this.service,
        "service.version": SERVICE_VERSION.startsWith("__")
          ? "dev"
          : SERVICE_VERSION,
        "deployment.environment": "akamai-functions",
        "cloud.provider": "akamai",
        "cloud.platform": "akamai_functions",
      }),
    };
  }

  private async post(path: string, body: any) {
    if (!ENABLED) return;
    try {
      await fetch(`${OTEL_ENDPOINT}${path}`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${OTEL_BEARER_TOKEN}`,
        },
        body: JSON.stringify(body),
        signal: AbortSignal.timeout(2000),
      });
    } catch (e) {
      // Telemetry must never break the request.
      console.error(`otel ${path} export failed: ${e}`);
    }
  }

  /** Flush all buffered spans + metrics. Call once at end of request. */
  async flush() {
    if (!ENABLED) return;
    const tasks: Promise<void>[] = [];

    if (this.spans.length > 0) {
      tasks.push(this.post("/v1/traces", {
        resourceSpans: [{
          resource: this.resource(),
          scopeSpans: [{
            scope: { name: "spin-otel", version: "1.0.0" },
            spans: this.spans.map((s) => ({
              traceId: s.traceId,
              spanId: s.spanId,
              parentSpanId: s.parentSpanId,
              name: s.name,
              kind: s.kind,
              startTimeUnixNano: s.startNano,
              endTimeUnixNano: s.endNano,
              attributes: kv(s.attrs),
              status: s.error ? { code: 2 } : { code: 1 },
            })),
          }],
        }],
      }));
    }

    if (this.counters.size > 0 || this.histos.size > 0) {
      const metrics: any[] = [];
      for (const [name, points] of this.counters) {
        metrics.push({
          name,
          unit: "1",
          sum: {
            aggregationTemporality: 2, // CUMULATIVE
            isMonotonic: true,
            dataPoints: points.map((p) => ({
              asInt: p.asInt,
              timeUnixNano: nowNanos(),
              attributes: kv(p.attrs),
            })),
          },
        });
      }
      for (const [name, points] of this.histos) {
        // Emit as a gauge-of-observations is wrong for histograms; instead
        // send each observation as an explicit-bucket histogram with a
        // single count. The collector aggregates across data points.
        metrics.push({
          name,
          unit: "ms",
          histogram: {
            aggregationTemporality: 2,
            dataPoints: points.map((p) => ({
              count: 1,
              sum: p.value,
              timeUnixNano: nowNanos(),
              attributes: kv(p.attrs),
              bucketCounts: [],
              explicitBounds: [],
            })),
          },
        });
      }
      tasks.push(this.post("/v1/metrics", {
        resourceMetrics: [{
          resource: this.resource(),
          scopeMetrics: [{
            scope: { name: "spin-otel", version: "1.0.0" },
            metrics,
          }],
        }],
      }));
    }

    await Promise.all(tasks);
  }
}

export function makeTracer(service: string): Tracer {
  return new Tracer(service);
}
