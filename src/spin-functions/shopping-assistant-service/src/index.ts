import { AutoRouter, cors, error, json } from 'itty-router';
import { makeTracer } from './otel';

const SERVICE = 'shopping-assistant-service';

// Zuplo AI Gateway endpoint (proxies to the upstream Gemma LLM).
// API key is replaced by CI at build time via sed substitution against
// the __ZUPLO_API_KEY__ placeholder (sourced from the ZUPLO_API_KEY
// GitHub Secret). Never commit the real key to git.
const LLM_ENDPOINT = "https://chat-ai-ai-gateway-34777e2.zuplo.app";
const ZUPLO_API_KEY = "__ZUPLO_API_KEY__";
const MODEL = "google_gemma-4-26B-A4B-it-Q4_K_M.gguf";

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
          'llm.endpoint': 'chat-ai-ai-gateway-34777e2.zuplo.app',
          'llm.model': MODEL,
        }, async (llmSpan) => {
          // Cache-buster: the Zuplo AI Gateway was caching completions with a
          // body-insensitive key, so every chat returned the first cached
          // answer (and the Firewall-for-AI block appeared bypassed). Force a
          // cache miss per request via a unique query param + no-cache headers
          // so each prompt gets a fresh completion.
          const nocache = `${Date.now()}-${Math.random().toString(36).slice(2)}`;
          const response = await fetch(`${LLM_ENDPOINT}/v1/chat/completions?nocache=${nocache}`, {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
              'Authorization': `Bearer ${ZUPLO_API_KEY}`,
              'Cache-Control': 'no-cache, no-store',
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

          const content: string = data.choices?.[0]?.message?.content ?? '';
          return json({ message: content.trim() });
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
