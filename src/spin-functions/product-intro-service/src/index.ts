import { AutoRouter, cors, error, json } from 'itty-router';

// ---- Product Catalog (embedded) ----
interface Product {
  id: string;
  name: string;
  description: string;
  price: number;
  categories: string[];
}

const PRODUCTS: Product[] = [
  { id: "AKMT001", name: "Akamai Unisex Short Sleeve Tee - Blue",     description: "Rep the edge in style with this Akamai \"We Move Mountains\" crew-neck tee in blue. Made from soft, breathable cotton blend.",                                                                    price: 28,  categories: ["apparel", "t-shirts"] },
  { id: "AKMT002", name: "Akamai Unisex Short Sleeve Tee - Gray",     description: "The Akamai \"We Move Mountains\" tee in heather gray. Made from soft cotton blend with the iconic mountain graphic.",                                                                                       price: 28,  categories: ["apparel", "t-shirts"] },
  { id: "AKMT003", name: "Akamai Women's V-Neck Tee",                  description: "The Akamai \"We Move Mountains\" tee in a flattering women's v-neck cut in blue. Lightweight, soft cotton with a relaxed fit.",                                                                          price: 28,  categories: ["apparel", "t-shirts"] },
  { id: "AKMT004", name: "Akamai Unisex Short Sleeve Tee - Navy",     description: "A sleek navy tee featuring an artistic Akamai surf/wave graphic. Made from soft, breathable cotton.",                                                                                                       price: 28,  categories: ["apparel", "t-shirts"] },
  { id: "AKMT005", name: "Akamai Men's Adidas Polo - Navy",           description: "A premium Akamai x Adidas polo shirt in navy heather. Features moisture-wicking climalite fabric and the Akamai logo embroidered on the sleeve.",                                                         price: 55,  categories: ["apparel", "polo"] },
  { id: "AKMT006", name: "Akamai Women's Adidas Polo - Navy",         description: "The Akamai x Adidas polo in a tailored women's fit, navy heather. Features Adidas' moisture-wicking technology and the Akamai logo on the sleeve.",                                                       price: 55,  categories: ["apparel", "polo"] },
  { id: "AKMT007", name: "Akamai Men's Vest - Black/Gray",            description: "A premium insulated vest with the Akamai logo. Black upper with gray quilted lower, orange accent zippers. Lightweight warmth without bulk.",                                                              price: 65,  categories: ["apparel", "vests"] },
  { id: "AKMT008", name: "Akamai Women's Fleece Jacket - Navy",       description: "A cozy full-zip fleece jacket in navy with the Akamai logo. Features a stand-up collar, zippered pockets, and orange accent zippers.",                                                                    price: 75,  categories: ["apparel", "jackets"] },
  { id: "AKMT009", name: "Akamai Men's Fleece Jacket - Navy",         description: "A premium full-zip fleece jacket in navy with the Akamai logo. Features orange accent zippers, a zippered chest pocket, and a comfortable athletic fit.",                                                 price: 75,  categories: ["apparel", "jackets"] },
  { id: "AKMT010", name: "Akamai Pullover Hoodie - Navy",             description: "A classic navy pullover hoodie with the Akamai logo embroidered on the chest. Features a kangaroo pocket and adjustable drawstring hood.",                                                                price: 60,  categories: ["apparel", "hoodies"] },
  { id: "AKMT011", name: "Akamai Full Zip Hoodie - Charcoal",         description: "A stylish full-zip hoodie in charcoal with the Akamai logo. Features a kangaroo pocket and white drawstring accents.",                                                                                    price: 65,  categories: ["apparel", "hoodies"] },
  { id: "AKMT012", name: "Akamai Tech Socks",                          description: "Show off your Akamai pride from head to toe! These fun navy socks feature colorful Akamai tech-themed icons — clouds, servers, and robots in blue and orange.",                                          price: 12,  categories: ["accessories", "apparel"] },
  { id: "AKMT013", name: "Akamai x The North Face Trucker Cap",       description: "A premium collaboration between Akamai and The North Face. Features the Akamai logo embroidered on a structured black front with white mesh back. Adjustable snapback.",                                  price: 35,  categories: ["accessories", "hats"] },
  { id: "AKMT014", name: "Akamai Beach Towel - Blue (Dock & Bay)",    description: "A premium Akamai x Dock & Bay quick-dry beach towel in blue and white stripes. Made from 100% recycled materials, it dries 3x faster than a regular towel.",                                            price: 32,  categories: ["accessories", "sports"] },
  { id: "AKMT015", name: "Akamai Beach Towel - Orange (Dock & Bay)",  description: "A premium Akamai x Dock & Bay quick-dry beach towel in orange and white stripes. Made from 100% recycled materials. Compact, lightweight, and incredibly absorbent.",                                    price: 32,  categories: ["accessories", "sports"] },
  { id: "AKMT016", name: "Akamai Canvas Tote Bag",                    description: "A classic cream canvas tote bag with navy accents and the Akamai logo. Sturdy reinforced handles and a spacious main compartment. Eco-friendly and stylish for everyday use.",                           price: 22,  categories: ["accessories", "bags"] },
  { id: "AKMT017", name: "Akamai 40oz Tumbler with Handle",           description: "A massive 40oz insulated tumbler with handle and straw in slate blue, featuring the Akamai logo. Keeps drinks cold for 24+ hours.",                                                                       price: 40,  categories: ["accessories", "drinkware"] },
  { id: "AKMT018", name: "Akamai Notebook",                            description: "A premium navy hardcover notebook with the Akamai logo debossed on the cover. Features an elastic closure band, ribbon bookmark, and 160 lined pages.",                                                  price: 18,  categories: ["accessories", "stationery"] },
  { id: "AKMT019", name: "Akamai Stylus Pen",                          description: "A sleek silver stylus pen with the Akamai logo. Features a precision tip for touchscreens and a ballpoint pen on the other end.",                                                                        price: 12,  categories: ["accessories", "stationery"] },
  { id: "AKMT020", name: "Akamai PopSocket",                           description: "A sleek black PopSocket with the Akamai logo. Attaches to the back of your phone or case to provide a secure grip and a stand for hands-free viewing.",                                                  price: 15,  categories: ["accessories", "tech"] },
  { id: "AKMT021", name: "Akamai Golf Balls",                          description: "Premium Akamai-branded golf balls in a stylish navy gift box. Features the Akamai logo and golfer silhouettes. A perfect gift for the golf-loving tech professional. Set of 3.",                        price: 25,  categories: ["accessories", "sports"] },
  { id: "AKMT022", name: "Akamai Phone Stand & Ring Light",           description: "Elevate your video calls with this Akamai-branded phone stand and clip-on ring light combo. Adjustable brightness for perfect lighting in any environment.",                                              price: 22,  categories: ["accessories", "tech"] },
  { id: "AKMT023", name: "Akamai 25th Anniversary Cooler Bag",        description: "Celebrate 25 Years of One Akamai with this premium insulated cooler bag in black. Features the commemorative 25th anniversary logo. Holds 6+ cans with a shoulder strap.",                              price: 38,  categories: ["accessories", "bags"] },
  { id: "AKMT024", name: "Akamai 25th Anniversary Tumbler",           description: "Mark 25 Years of One Akamai with this sleek matte black insulated tumbler. Laser-engraved with the commemorative anniversary logo.",                                                                      price: 35,  categories: ["accessories", "drinkware"] },
  { id: "AKMT025", name: "Akamai 25th Anniversary Cap",               description: "A premium baseball cap celebrating 25 Years of One Akamai. Black structured cap with the colorful anniversary logo embroidered on the front and an orange-lined brim.",                                  price: 28,  categories: ["accessories", "hats"] },
  { id: "AKMT026", name: "Akamai Laptop Backpack",                    description: "A sleek gray and black Akamai laptop backpack with orange accent zippers. Features a padded laptop compartment, front zippered pocket, and top handle.",                                                  price: 55,  categories: ["accessories", "bags"] },
  { id: "AKMT027", name: "Akamai Gift Card - $50",                    description: "Give the gift of Akamai swag! This gift card can be used to purchase any item in the Akamai store.",                                                                                                     price: 50,  categories: ["gift-cards"] },
  { id: "AKMT028", name: "PEACE FOR ALL Tee / Akamai - White",        description: "Over 25 years ago, Akamai laid the foundation for today's connected internet. This T-shirt is a tribute to the dawn of the internet era. The beige color references the iconic 'beige box' computers of that time. The heart on the chest symbolizes how the internet has connected people toward a better world. The code on the back is real — a nod to Linux and the open-source spirit powering the web.",                                                                          price: 29.99, categories: ["apparel", "t-shirts"] },
  { id: "AKMT029", name: "PEACE FOR ALL Tee / Akamai - Black",        description: "Every day, billions of people connect online — working, playing, learning, shopping, and sharing ideas. The 'texture' code featured in this design represents the common language behind every digital experience. For 25 years, Akamai has built a comfortable, secure internet to enrich lives around the world. We remain committed to making the world safer and more deeply connected.",                                                                                             price: 29.99, categories: ["apparel", "t-shirts"] },
];

const LLM_ENDPOINT = "http://172.238.48.187:8000";

// ---- LLM call ----
async function generateIntro(product: Product, lang: string): Promise<string> {
  const isJapanese = lang === 'ja';

  const prompt = isJapanese
    ? `あなたはAkamaiストアの商品紹介ライターです。
以下の商品について、魅力的な紹介文を日本語で2〜3文で書いてください。
価格帯（$${product.price}）に見合った価値を伝え、購買意欲を高める内容にしてください。
紹介文のみを返してください。余分な説明は不要です。

商品名: ${product.name}
カテゴリ: ${product.categories.join(", ")}
元の説明: ${product.description}`
    : `You are a product copywriter for the Akamai store.
Write a compelling 2-3 sentence product introduction in English for the following item.
Convey the value for the price ($${product.price}) and inspire the customer to buy.
Return only the introduction text, nothing else.

Product: ${product.name}
Category: ${product.categories.join(", ")}
Description: ${product.description}`;

  try {
    const response = await fetch(`${LLM_ENDPOINT}/v1/chat/completions`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        model: "google_gemma-4-26B-A4B-it-Q4_K_M.gguf",
        messages: [{ role: "user", content: prompt }],
        max_tokens: 300,
        temperature: 0.7,
      }),
    });

    if (!response.ok) {
      console.error(`LLM request failed: ${response.status}`);
      return product.description;
    }

    const data = await response.json() as any;
    const content: string = data.choices?.[0]?.message?.content ?? "";
    return content.trim() || product.description;
  } catch (e) {
    console.error(`Error calling LLM: ${e}`);
    return product.description;
  }
}

// ---- Router ----
const { preflight, corsify } = cors({ origin: '*' });

const router = AutoRouter({
  before: [preflight],
  finally: [corsify],
});

router
  .get('/healthz', () => json({ status: 'ok' }))
  .get('/intro', async (req: Request) => {
    const url = new URL(req.url);
    const productId = url.searchParams.get('product_id');
    const lang = url.searchParams.get('lang') ?? 'en';

    if (!productId) {
      return error(400, { error: 'product_id query parameter is required' });
    }

    const product = PRODUCTS.find(p => p.id === productId);
    if (!product) {
      return error(404, { error: `Product ${productId} not found` });
    }

    if (!['en', 'ja'].includes(lang)) {
      return error(400, { error: 'lang must be "en" or "ja"' });
    }

    const intro = await generateIntro(product, lang);
    return json({
      product_id: productId,
      product_name: product.name,
      lang,
      intro,
    });
  });

//@ts-ignore
addEventListener('fetch', (event: FetchEvent) => {
  event.respondWith(router.fetch(event.request));
});
