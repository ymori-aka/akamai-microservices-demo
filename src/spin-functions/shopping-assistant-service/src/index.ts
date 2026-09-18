import { AutoRouter, cors, error, json } from 'itty-router';
import { makeTracer } from './otel';

const SERVICE = 'shopping-assistant-service';

// Zuplo AI Gateway endpoint (proxies to the upstream Gemma LLM).
// API key is replaced by CI at build time via sed substitution against
// the __ZUPLO_API_KEY__ placeholder (sourced from the ZUPLO_API_KEY
// GitHub Secret). Never commit the real key to git.
//
// Migrated to the new gateway 2026-09-02 (the old chat-ai-ai-gateway-34777e2
// endpoint now 500s). Two things this gateway requires that the old one
// didn't:
//   - The path must include /config_<id> — the host alone 404s with
//     "Unsupported AI Gateway endpoint".
//   - `model` must be "providerName/model", not the bare filename — a bare
//     model name 400s. The provider configured for Gemma here is "gemma4".
const LLM_ENDPOINT = "https://ec-chat-main-c4d1d89.zuplo.app/config_3114835392f54da99555e30c95cd15a6";
const ZUPLO_API_KEY = "__ZUPLO_API_KEY__";
const MODEL = "gemma4/google_gemma-4-26B-A4B-it-Q4_K_M.gguf";

const { preflight, corsify } = cors({ origin: '*' });

const router = AutoRouter({
  before: [preflight],
  finally: [corsify],
});

router
  .get('/healthz', () => json({ status: 'ok' }))

  // POST /chat
  // Request body: { messages: [{role, content}][], max_tokens?, temperature? }
  // Response:     { message: string }
  .post('/chat', async (req: Request) => {
    const tracer = makeTracer(SERVICE);
    const start = Date.now();
    const route = 'POST /chat';
    let statusCode = 200;

    const result = await tracer.withSpan(route, 'SERVER', {
      'http.method': 'POST',
      'http.route': '/chat',
    }, async (serverSpan) => {
      let body: any;
      try {
        body = await req.json();
      } catch {
        statusCode = 400;
        return error(400, { error: 'Invalid JSON body' });
      }

      if (!body.messages || !Array.isArray(body.messages)) {
        statusCode = 400;
        return error(400, { error: 'messages array is required' });
      }

      const parentId = tracer.lastSpanId();
      try {
        return await tracer.withSpan('llm.chat.completions', 'CLIENT', {
          'llm.endpoint': 'ec-chat-main-c4d1d89.zuplo.app',
          'llm.model': MODEL,
        }, async (llmSpan) => {
          // The gateway used to cache completions with a body-insensitive key
          // (semanticTolerance was 0.4, i.e. anything above 0.6 similarity hit),
          // so every chat returned the first cached answer and the
          // Firewall-for-AI block appeared bypassed. That is fixed on the
          // gateway side, so the semantic cache is now left enabled and its
          // outcome is surfaced in the UI. Send `"nocache": true` in the body
          // to force a fresh completion for a particular request.
          const bustCache = body.nocache === true;
          const url = bustCache
            ? `${LLM_ENDPOINT}/v1/chat/completions?nocache=${Date.now()}-${Math.random().toString(36).slice(2)}`
            : `${LLM_ENDPOINT}/v1/chat/completions`;
          const response = await fetch(url, {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
              'Authorization': `Bearer ${ZUPLO_API_KEY}`,
              ...(bustCache ? { 'Cache-Control': 'no-cache, no-store' } : {}),
            },
            body: JSON.stringify({
              model: MODEL,
              messages: body.messages,
              max_tokens: body.max_tokens ?? 512,
              temperature: body.temperature ?? 0.7,
            }),
          });

          llmSpan.setAttr('http.status_code', response.status);

          if (!response.ok) {
            console.error(`LLM returned ${response.status}`);
            statusCode = 502;
            tracer.recordCounter('spin_llm_errors_total', 1, {
              service: SERVICE, model: MODEL, reason: `http_${response.status}`,
            });
            return error(502, { error: `LLM request failed: ${response.status}` });
          }

          const data = await response.json() as any;
          const usage = data.usage ?? {};
          if (usage.prompt_tokens) {
            llmSpan.setAttr('llm.tokens.prompt', usage.prompt_tokens);
            tracer.recordCounter('spin_llm_tokens_total', usage.prompt_tokens, {
              service: SERVICE, model: MODEL, kind: 'prompt',
            });
          }
          if (usage.completion_tokens) {
            llmSpan.setAttr('llm.tokens.completion', usage.completion_tokens);
            tracer.recordCounter('spin_llm_tokens_total', usage.completion_tokens, {
              service: SERVICE, model: MODEL, kind: 'completion',
            });
          }
          if (usage.total_tokens) llmSpan.setAttr('llm.tokens.total', usage.total_tokens);

          // Smart Router's classification only exists on the gateway side, so
          // ec-chat's smart-router-headers-outbound policy copies it onto
          // x-ai-* response headers. Pass it through so the chat UI can show
          // which tier the prompt landed in and which model answered.
          const h = response.headers;
          const complexity = h.get('x-ai-complexity');
          const routing = complexity
            ? {
                intent: h.get('x-ai-intent'),
                complexity,
                confidence: Number(h.get('x-ai-confidence')),
                applied: h.get('x-ai-routing-applied') === 'true',
                reason: h.get('x-ai-routing-reason'),
                classify_ms: Number(h.get('x-ai-classify-ms')),
                // What Smart Router picked; falls back to whatever the gateway
                // actually served when routing did not apply.
                model: h.get('x-ai-routed-model') ?? data.model ?? null,
                provider: data.provider ?? null,
              }
            : null;

          if (routing) {
            llmSpan.setAttr('llm.routed.intent', routing.intent ?? '');
            llmSpan.setAttr('llm.routed.complexity', routing.complexity);
            llmSpan.setAttr('llm.routed.model', routing.model ?? '');
          }

          // Semantic cache outcome (RFC 9211 Cache-Status plus Zuplo's own
          // headers). A HIT means no GPU produced this answer, so the UI shows
          // it explicitly rather than letting a cached reply look like a fresh
          // generation.
          const cacheState = h.get('x-ai-gateway-cache');
          const cache = cacheState
            ? {
                state: cacheState.toUpperCase(),
                similarity: Number(h.get('x-ai-gateway-cache-similarity')),
                status: h.get('cache-status'),
              }
            : null;
          if (cache) {
            llmSpan.setAttr('llm.cache.state', cache.state);
            if (isFinite(cache.similarity)) llmSpan.setAttr('llm.cache.similarity', cache.similarity);
            tracer.recordCounter('spin_llm_cache_total', 1, {
              service: SERVICE, model: MODEL, state: cache.state,
            });
          }

          const content: string = data.choices?.[0]?.message?.content ?? '';
          // The frontend renders these next to the answer (tokens / finish
          // reason / cache), so forward what the gateway reported.
          const finish = data.choices?.[0]?.finish_reason ?? null;
          return json({ message: content.trim(), routing, cache, usage, finish });
        }, parentId);
      } catch (e) {
        console.error(`Error calling LLM: ${e}`);
        statusCode = 502;
        tracer.recordCounter('spin_llm_errors_total', 1, {
          service: SERVICE, model: MODEL, reason: 'fetch_failed',
        });
        return error(502, { error: `Failed to reach LLM: ${e}` });
      } finally {
        serverSpan.setAttr('http.status_code', statusCode);
      }
    });

    // Metrics + flush (telemetry failures never affect the response).
    tracer.recordCounter('spin_requests_total', 1, {
      service: SERVICE, route, status_code: statusCode,
    });
    tracer.recordHistogram('spin_request_duration_ms', Date.now() - start, {
      service: SERVICE, route, status_code: statusCode,
    });
    await tracer.flush();

    return result;
  });

//@ts-ignore
addEventListener('fetch', (event: FetchEvent) => {
  event.respondWith(router.fetch(event.request));
});
