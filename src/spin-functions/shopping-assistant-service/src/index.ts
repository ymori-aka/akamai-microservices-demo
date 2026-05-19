import { AutoRouter, cors, error, json } from 'itty-router';

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
    let body: any;
    try {
      body = await req.json();
    } catch {
      return error(400, { error: 'Invalid JSON body' });
    }

    if (!body.messages || !Array.isArray(body.messages)) {
      return error(400, { error: 'messages array is required' });
    }

    try {
      const response = await fetch(`${LLM_ENDPOINT}/v1/chat/completions`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${ZUPLO_API_KEY}`,
        },
        body: JSON.stringify({
          model: MODEL,
          messages: body.messages,
          max_tokens: body.max_tokens ?? 512,
          temperature: body.temperature ?? 0.7,
        }),
      });

      if (!response.ok) {
        console.error(`LLM returned ${response.status}`);
        return error(502, { error: `LLM request failed: ${response.status}` });
      }

      const data = await response.json() as any;
      const content: string = data.choices?.[0]?.message?.content ?? '';
      return json({ message: content.trim() });

    } catch (e) {
      console.error(`Error calling LLM: ${e}`);
      return error(502, { error: `Failed to reach LLM: ${e}` });
    }
  });

//@ts-ignore
addEventListener('fetch', (event: FetchEvent) => {
  event.respondWith(router.fetch(event.request));
});
