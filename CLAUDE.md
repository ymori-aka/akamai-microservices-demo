# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this repository is

This is **"Akamai Microservices Demo"**, a heavily customized fork of
[GoogleCloudPlatform/microservices-demo](https://github.com/GoogleCloudPlatform/microservices-demo)
(Online Boutique). It is a live demo environment, not a general-purpose open
source project: it runs on a single Linode Kubernetes Engine (LKE) cluster,
`jp-tyo-3` / Tokyo, cluster id 610031, serving the demo at
`aka-store.tserof.net`. Pushes to `main` auto-deploy to it via GitHub
Actions. (An earlier Osaka cluster, `jp-osa`, id 598466, was decommissioned
by the user — don't assume a second cluster exists or that the self-hosted
runner's default kubeconfig points anywhere but Tokyo now; verify before
relying on old assumptions.) Read `README.md` (English) / `README.ja.md`
(Japanese) first — they document the full architecture, data stores,
observability stack, and setup steps in detail and are kept up to date;
don't duplicate that content from memory.

**Branch layout:** `main` = Redis-backed cart (original). `valkey` = fork
using Akamai Managed Valkey (Aiven, beta) instead of in-cluster Redis for
the cart store and a sales-ranking feature (Valkey Sorted Set). The two are
intentionally kept separate — do not merge `valkey` into `main` without the
user asking, and do not let branch-specific app changes (Redis vs Valkey
connection code, image tags, workflows) leak across branches.

## Repository structure (what's genuinely non-obvious)

- `src/<service>/` — one directory per microservice, in its original
  upstream language (Go / Python / Node.js / Java / C#). Each has its own
  `Dockerfile` and, where gRPC is used, a `genproto.sh` that regenerates
  stubs from `protos/demo.proto` into a local `genproto/` directory.
- `src/frontend/` is the **primary modified service** — most demo-specific
  logic lives here: `handlers.go` (routing + admin API), `orders_db.go`
  (Postgres reads for `/orders`, `/admin/orders`), `translations.go`
  (ja/ko/zh strings), `deployment_details.go`, `packaging_info.go`,
  `templates/` (server-rendered HTML).
- `src/checkoutservice/orders_db.go` — writes placed orders to Postgres
  (non-blocking: checkout succeeds even if the DB write fails).
- `src/productcatalogservice/catalog_loader_mongo.go` + `products.json` —
  MongoDB-backed catalog; self-seeds from `products.json` on first start
  if the collection is empty, then reads from Mongo only.
- `src/spin-functions/*` — three separate TypeScript Fermyon Spin apps
  (`recommendation-service`, `product-intro-service`,
  `shopping-assistant-service`) deployed independently to **Akamai
  Functions** (edge, not the k8s cluster). Each has its own
  `package.json` / `build.mjs` / `spin.toml`.
- `kubernetes-manifests/` — plain manifests (no Helm/Kustomize overlays
  used in practice beyond the base `kustomization.yaml`), applied directly
  with `kubectl apply -f` by CI. `monitoring/` holds the full in-cluster
  observability stack (Prometheus, Loki, Tempo, Grafana, `aclp-collector`
  for Akamai Cloud Pulse, `postgres-exporter` for business KPIs).
- `helm-chart/`, `kustomize/`, `terraform/`, `terraform-linode/` — mostly
  vestigial from upstream/earlier iterations; the actual deploy path is the
  GitHub Actions workflows below, not these.
- `.github/workflows/` — dozens of workflows beyond the main CI/CD pipeline
  (`deploy.yml`, `deploy-tokyo.yml`): many are one-off `fix-*` / `diag-*` /
  `chk-*` operational scripts written to diagnose and repair live cluster
  incidents (Calico/BGP, istiod webhook blocking rollouts, Tempo PVC
  corruption, ACLP collector token expiry, etc.). Treat these as evidence
  of past incidents, not as generic reusable tooling — read the specific
  workflow before assuming what it does.

## Local development

Follow `docs/development-guide.md` for the full `skaffold`-based local dev
loop against a real (or Minikube) Kubernetes cluster. Key points:

```bash
skaffold run    # one-time build + deploy of every service in skaffold.yaml
skaffold dev    # continuous build + deploy + log tailing on file changes
```

`skaffold.yaml` defines the artifact list — the image name/build context
per service — and delegates manifests to `kubernetes-manifests/` via
Kustomize. There's also a `debug` profile (swaps in `Dockerfile.debug` for
cartservice) and a `network-policies` profile.

### Per-service build/test (when iterating on one service without Skaffold)

- **Go services** (`frontend`, `checkoutservice`, `productcatalogservice`,
  `shippingservice`): standard `go build ./...` / `go test ./...` from
  inside the service directory. Existing unit tests: `money/money_test.go`
  (frontend, checkoutservice), `product_catalog_test.go`
  (productcatalogservice), `shippingservice_test.go`, `validator/validator_test.go`
  (frontend). Run a single test with `go test ./money/ -run TestSomething -v`.
  Regenerate protobuf stubs with `./genproto.sh` (requires `protoc` +
  `protoc-gen-go` / `protoc-gen-go-grpc` on `PATH`).
- **cartservice** (C#/.NET): solution is `src/cartservice/cartservice.sln`;
  build with `dotnet build`, test with `dotnet test` (tests live under
  `src/cartservice/tests`).
- **Node.js services** (`currencyservice`, `paymentservice`): `npm install`
  then check `package.json` scripts; regenerate protos with `genproto.sh`.
- **Python services** (`recommendationservice`, `emailservice`,
  `loadgenerator`, `shoppingassistantservice`): dependencies pinned in
  `requirements.in` → compiled to `requirements.txt`.
- **adservice** (Java/Gradle): `./gradlew build` from `src/adservice`.
- **spin-functions/*** (TypeScript): `npm install && npm run build` in each
  service directory (uses `build.mjs`, esbuild-style bundling for Wasm).

## CI/CD reality

- `.github/workflows/deploy.yml` — triggers on push to `main`. Builds
  frontend/currencyservice/productcatalogservice/checkoutservice/cartservice
  images in parallel on GitHub-hosted runners, tags `sha-<short-sha>` (and
  `latest` on `main`), pushes to `ghcr.io/ymori-aka/<service>`, then a
  **self-hosted runner** job applies manifests and rolls out via whatever
  kubeconfig is currently the runner's default. This historically pointed
  at the now-deleted Osaka cluster — confirm what it targets today before
  assuming a `main` push alone redeploys the live Tokyo demo.
- `.github/workflows/deploy-tokyo.yml` — the Tokyo (jp-tyo-3, 610031)
  equivalent; the demo's actual public deployment lives here. It fetches
  the Tokyo kubeconfig explicitly rather than relying on the runner's
  default context, so it's the reliable path regardless of the runner's
  default.
- Feature-branch workflows are **not** dispatchable via
  `workflow_dispatch --ref <branch>` until that workflow file also exists
  on `main` (GitHub only registers workflows it can see on the default
  branch). The working pattern used repeatedly in this repo: temporarily
  add/copy the workflow to `main`, dispatch it with `--ref <branch>`, then
  remove it from `main` once done — don't leave branch-specific CI/deploy
  workflows permanently registered on `main`.
- This repo is **public** — never print secrets (Linode PAT, DB DSNs,
  Valkey credentials, GHCR tokens) in workflow logs or commit them into
  manifests/scripts.

## コミュニケーション

- 日本語で応答する(コード・変数名は英語)
- 簡潔に回答し、自明な説明は省略する
- 複雑なタスクでは実装前に計画を提示し、承認後に着手する

## コードスタイル

- 関数型アプローチを優先し、副作用を最小化する
- 厳密な型付け(anyは使わずunknownを使う)
- エラーは握りつぶさず、意味のあるメッセージ付きで処理する

## Git規約

- Conventional Commits形式、本文は日本語(例: `feat: ユーザー認証にOAuth2を追加`)
- 確認なしに自動コミット・自動pushしない

## 禁止事項

- README・ドキュメントを勝手に生成・変更しない
- テストコードを確認なしに削除・コメントアウトしない
- 既存の動作するコードを理由なくリファクタリングしない
