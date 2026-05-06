import { AutoRouter, cors, error, json } from 'itty-router';

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

// ---- LLM call ----
async function getRecommendations(productId: string): Promise<string[]> {
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

  try {
    const response = await fetch(`${LLM_ENDPOINT}/v1/chat/completions`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        model: "google_gemma-4-26B-A4B-it-Q4_K_M.gguf",
        messages: [{ role: "user", content: prompt }],
        max_tokens: 100,
        temperature: 0.3,
      }),
    });

    if (!response.ok) {
      console.error(`LLM request failed: ${response.status}`);
      return getFallbackRecommendations(productId);
    }

    const data = await response.json() as any;
    const content: string = data.choices?.[0]?.message?.content ?? "";

    // Extract JSON array from response
    const match = content.match(/\[.*?\]/s);
    if (!match) return getFallbackRecommendations(productId);

    const ids: string[] = JSON.parse(match[0]);
    // Validate IDs exist in catalog
    return ids.filter(id => PRODUCTS.some(p => p.id === id)).slice(0, 4);
  } catch (e) {
    console.error(`Error calling LLM: ${e}`);
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
    const url = new URL(req.url);
    const productId = url.searchParams.get('product_id');

    if (!productId) {
      return error(400, { error: 'product_id query parameter is required' });
    }

    const product = PRODUCTS.find(p => p.id === productId);
    if (!product) {
      return error(404, { error: `Product ${productId} not found` });
    }

    const recommendations = await getRecommendations(productId);
    return json({ product_id: productId, recommendations });
  });

//@ts-ignore
addEventListener('fetch', (event: FetchEvent) => {
  event.respondWith(router.fetch(event.request));
});
