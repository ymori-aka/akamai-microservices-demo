import { AutoRouter, cors, error, json } from 'itty-router';
import { makeTracer, Tracer } from './otel';

const SERVICE = 'recommendation-service';
const MODEL = 'google_gemma-4-26B-A4B-it-Q4_K_M.gguf';

// ---- Product Catalog (embedded) ----
const PRODUCTS = [
  { id: "AKMT001", name: "Akamai Unisex Short Sleeve Tee - Blue",     categories: ["apparel", "t-shirts"] },
  { id: "AKMT002", name: "Akamai Unisex Short Sleeve Tee - Gray",     categories: ["apparel", "t-shirts"] },
  { id: "AKMT003", name: "Akamai Women's V-Neck Tee",                  categories: ["apparel", "t-shirts"] },
  { id: "AKMT004", name: "Akamai Unisex Short Sleeve Tee - Navy",     categories: ["apparel", "t-shirts"] },
  { id: "AKMT005", name: "Akamai Men's Adidas Polo - Navy",           categories: ["apparel", "polo"] },
  { id: "AKMT006", name: "Akamai Women's Adidas Polo - Navy",         categories: ["apparel", "polo"] },
  { id: "AKMT007", name: "Akamai Men's Vest - Black/Gray",            categories: ["apparel", "vests"] },
  { id: "AKMT008", name: "Akamai Women's Fleece Jacket - Navy",       categories: ["apparel", "jackets"] },
  { id: "AKMT009", name: "Akamai Men's Fleece Jacket - Navy",         categories: ["apparel", "jackets"] },
  { id: "AKMT010", name: "Akamai Pullover Hoodie - Navy",             categories: ["apparel", "hoodies"] },
  { id: "AKMT011", name: "Akamai Full Zip Hoodie - Charcoal",         categories: ["apparel", "hoodies"] },
  { id: "AKMT012", name: "Akamai Tech Socks",                          categories: ["accessories", "apparel"] },
  { id: "AKMT013", name: "Akamai x The North Face Trucker Cap",       categories: ["accessories", "hats"] },
  { id: "AKMT014", name: "Akamai Beach Towel - Blue (Dock & Bay)",    categories: ["accessories", "sports"] },
  { id: "AKMT015", name: "Akamai Beach Towel - Orange (Dock & Bay)",  categories: ["accessories", "sports"] },
  { id: "AKMT016", name: "Akamai Canvas Tote Bag",                    categories: ["accessories", "bags"] },
  { id: "AKMT017", name: "Akamai 40oz Tumbler with Handle",           categories: ["accessories", "drinkware"] },
  { id: "AKMT018", name: "Akamai Notebook",                            categories: ["accessories", "stationery"] },
  { id: "AKMT019", name: "Akamai Stylus Pen",                          categories: ["accessories", "stationery"] },
  { id: "AKMT020", name: "Akamai PopSocket",                           categories: ["accessories", "tech"] },
  { id: "AKMT021", name: "Akamai Golf Balls",                          categories: ["accessories", "sports"] },
  { id: "AKMT022", name: "Akamai Phone Stand & Ring Light",           categories: ["accessories", "tech"] },
  { id: "AKMT023", name: "Akamai 25th Anniversary Cooler Bag",        categories: ["accessories", "bags"] },
  { id: "AKMT024", name: "Akamai 25th Anniversary Tumbler",           categories: ["accessories", "drinkware"] },
  { id: "AKMT025", name: "Akamai 25th Anniversary Cap",               categories: ["accessories", "hats"] },
  { id: "AKMT026", name: "Akamai Laptop Backpack",                    categories: ["accessories", "bags"] },
  { id: "AKMT027", name: "Akamai Gift Card - $50",                    categories: ["gift-cards"] },
  { id: "AKMT028", name: "PEACE FOR ALL Tee / Akamai - White",        categories: ["apparel", "t-shirts"] },
  { id: "AKMT029", name: "PEACE FOR ALL Tee / Akamai - Black",        categories: ["apparel", "t-shirts"] },
];

const LLM_ENDPOINT = "http://172.238.48.187:8000";

// Injected at build time by the deploy workflow, which substitutes the
// __GEMMA_API_KEY__ placeholder from the GEMMA_API_KEY GitHub Secret.
// The Gemma server runs with --api_key, so requests without this header get
// a 401 and the handler silently falls back to a non-LLM result.
const GEMMA_API_KEY = "__GEMMA_API_KEY__";

// ---- LLM call ----
async function getRecommendations(productId: string, tracer?: Tracer): Promise<string[]> {
  const product = PRODUCTS.find(p => p.id === productId);
  if (!product) return [];

  const otherProducts = PRODUCTS.filter(p => p.id !== productId);
  const catalogList = otherProducts.map(p => `- ${p.id}: ${p.name} (${p.categories.join(", ")})`).join("\n");

  const prompt = `You are a product recommendation engine for the Akamai store.

A customer is viewing: "${product.name}" (categories: ${product.categories.join(", ")})

From the following product catalog, recommend exactly 4 products that this customer would likely be interested in.
Return ONLY a JSON array of product IDs, nothing else. Example: ["AKMT002","AKMT005","AKMT010","AKMT013"]

Product catalog:
${catalogList}`;

  const callLLM = async (llmSpan?: { setAttr: (k: string, v: any) => void }) => {
    const response = await fetch(`${LLM_ENDPOINT}/v1/chat/completions`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "Authorization": `Bearer ${GEMMA_API_KEY}`,
      },
      body: JSON.stringify({
        model: MODEL,
        messages: [{ role: "user", content: prompt }],
        max_tokens: 100,
        temperature: 0.3,
      }),
    });

    llmSpan?.setAttr('http.status_code', response.status);

    if (!response.ok) {
      console.error(`LLM request failed: ${response.status}`);
      tracer?.recordCounter('spin_llm_errors_total', 1, {
        service: SERVICE, model: MODEL, reason: `http_${response.status}`,
      });
      return getFallbackRecommendations(productId);
    }

    const data = await response.json() as any;
    const usage = data.usage ?? {};
    if (usage.prompt_tokens) tracer?.recordCounter('spin_llm_tokens_total', usage.prompt_tokens, { service: SERVICE, model: MODEL, kind: 'prompt' });
    if (usage.completion_tokens) tracer?.recordCounter('spin_llm_tokens_total', usage.completion_tokens, { service: SERVICE, model: MODEL, kind: 'completion' });
    const content: string = data.choices?.[0]?.message?.content ?? "";

    // Extract JSON array from response
    const match = content.match(/\[.*?\]/s);
    if (!match) return getFallbackRecommendations(productId);

    const ids: string[] = JSON.parse(match[0]);
    // Validate IDs exist in catalog
    return ids.filter(id => PRODUCTS.some(p => p.id === id)).slice(0, 4);
  };

  try {
    if (tracer) {
      const parentId = tracer.lastSpanId();
      return await tracer.withSpan('llm.chat.completions', 'CLIENT', {
        'llm.endpoint': 'recommendation-llm', 'llm.model': MODEL,
      }, (llmSpan) => callLLM(llmSpan), parentId);
    }
    return await callLLM();
  } catch (e) {
    console.error(`Error calling LLM: ${e}`);
    tracer?.recordCounter('spin_llm_errors_total', 1, {
      service: SERVICE, model: MODEL, reason: 'fetch_failed',
    });
    return getFallbackRecommendations(productId);
  }
}

// Fallback: same category products
function getFallbackRecommendations(productId: string): string[] {
  const product = PRODUCTS.find(p => p.id === productId);
  if (!product) return PRODUCTS.slice(0, 4).map(p => p.id);

  const sameCategory = PRODUCTS.filter(
    p => p.id !== productId && p.categories.some(c => product.categories.includes(c))
  );
  const others = PRODUCTS.filter(
    p => p.id !== productId && !sameCategory.includes(p)
  );
  return [...sameCategory, ...others].slice(0, 4).map(p => p.id);
}

// ---- Router ----
const { preflight, corsify } = cors({ origin: '*' });

const router = AutoRouter({
  before: [preflight],
  finally: [corsify],
});

router
  .get('/healthz', () => json({ status: 'ok' }))
  .get('/recommendations', async (req: Request) => {
    const tracer = makeTracer(SERVICE);
    const start = Date.now();
    const route = 'GET /recommendations';
    let statusCode = 200;

    const result = await tracer.withSpan(route, 'SERVER', {
      'http.method': 'GET', 'http.route': '/recommendations',
    }, async (serverSpan) => {
      const url = new URL(req.url);
      const productId = url.searchParams.get('product_id');

      if (!productId) {
        statusCode = 400;
        serverSpan.setAttr('http.status_code', 400);
        return error(400, { error: 'product_id query parameter is required' });
      }

      const product = PRODUCTS.find(p => p.id === productId);
      if (!product) {
        statusCode = 404;
        serverSpan.setAttr('http.status_code', 404);
        return error(404, { error: `Product ${productId} not found` });
      }

      const recommendations = await getRecommendations(productId, tracer);
      serverSpan.setAttr('http.status_code', 200);
      return json({ product_id: productId, recommendations });
    });

    tracer.recordCounter('spin_requests_total', 1, { service: SERVICE, route, status_code: statusCode });
    tracer.recordHistogram('spin_request_duration_ms', Date.now() - start, { service: SERVICE, route, status_code: statusCode });
    await tracer.flush();

    return result;
  });

//@ts-ignore
addEventListener('fetch', (event: FetchEvent) => {
  event.respondWith(router.fetch(event.request));
});
