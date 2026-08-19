[English](./README.md) | **日本語**

# Akamai Microservices Demo

> **Akamai の各種サービスを組み合わせた、エンドツーエンドのデモ環境です。**
> LKE（Linode Kubernetes Engine）上のマイクロサービス EC サイトに、
> Akamai Functions によるエッジ AI、Linode のマネージドサービス
> （**Managed Valkey** を含む）によるデータ永続化、サービスメッシュと
> カオスエンジニアリング、クラスタ内 Grafana スタック + Akamai Cloud Pulse に
> よるフル可観測性を統合した構成です。

[![Build and Deploy](https://github.com/ymori-aka/akamai-microservices-demo/actions/workflows/deploy.yml/badge.svg)](https://github.com/ymori-aka/akamai-microservices-demo/actions/workflows/deploy.yml)

> **稼働環境:** 東京の LKE クラスタ 1 面（**`jp-tyo-3`, cluster id `610031`**）。
> 以前あった大阪（`jp-osa`）クラスタは廃止済みです。
>
> **ブランチ構成:** `main` は在クラスタ **Redis** のカートストアをそのまま維持し、
> **`valkey` ブランチ**がそれを **Akamai Managed Valkey** に置き換えて売れ筋
> ランキングを追加しています。**ライブデモにデプロイされているのは `valkey`
> ブランチ**なので、この README はその構成を前提に記述しています。`main` を
> checkout して `kubectl apply -f kubernetes-manifests/` を実行した場合は
> Redis 版になります。

---

## 目次

- [デモで見せられること](#デモで見せられること)
- [アーキテクチャ](#アーキテクチャ)
- [技術スタック](#技術スタック)
- [データ永続化](#データ永続化)
- [可観測性](#可観測性)
- [デモ環境へのアクセス](#デモ環境へのアクセス)
- [セットアップ手順](#セットアップ手順)
- [CI/CD パイプライン](#cicd-パイプライン)
- [商品管理画面の使い方](#商品管理画面の使い方)
- [リポジトリ構成](#リポジトリ構成)
- [ライセンス・派生元について](#ライセンス派生元について)

---

## デモで見せられること

| # | シナリオ | 訴求ポイント |
|---|----------|-------------|
| 1 | **EC サイトを LKE で運用** | マネージド Kubernetes の手軽さ、スケーラビリティ |
| 2 | **Akamai Functions で商品紹介を AI 生成** | エッジで動く Function から、GPU 上の LLM（Gemma 4）を低レイテンシで呼び出し |
| 3 | **AI によるパーソナライズドおすすめ** | 閲覧中の商品に応じた動的なレコメンド |
| 4 | **AI ショッピングアシスタントとチャット** | Spin Function 経由で Gemma にチャット問い合わせ |
| 5 | **チャット経路の Firewall for AI** | チャットのプロンプトは Zuplo AI Gateway を通過し、カード番号 / PII / 過大入力（DoS 様）を LLM 到達前に遮断 |
| 6 | **4 言語 UI** | 英語 / 日本語 / 한국어 / 中文 をワンクリックで切り替え |
| 7 | **8 通貨対応** | USD / EUR / JPY / CAD / GBP / TRY / KRW / CNY、ライブ為替レート |
| 8 | **注文履歴の永続化** | Linode Managed PostgreSQL に注文を保存、`/orders` / `/admin/orders` で閲覧 |
| 9 | **MongoDB ベースの商品カタログ** | StatefulSet + PVC、初回起動時に自動シード、管理画面から CRUD |
| 10 | **商品画像を Linode Object Storage で配信** | `*.linodeobjects.com` から画像を提供（クラスタ内配信ではなく） |
| 11 | **カートストアに Managed Valkey を採用** | 在クラスタ Redis を **Akamai Managed Valkey**（Aiven ベース）へ TLS 接続で置き換え。アプリの書き換えなしでマネージドサービスへ移行 |
| 12 | **Valkey によるライブ売れ筋ランキング** | 注文ごとに Valkey の **Sorted Set** へ `ZINCRBY`。ストアがリアルタイムのランキングを表示（ホームのサイドバー + `/ranking`）。Valkey が単なるキャッシュではないことを訴求 |
| 13 | **クラスタ内 Grafana + Cloud Pulse** | DB / Valkey / LB / LLM のメトリクスを 1 つのダッシュボードで表示。前日・当日の注文数と売上を PostgreSQL から直接クエリして表示 |
| 14 | **LLM のエンドツーエンド計装** | モデル別の token 使用量、レイテンシ p50/p95/p99、エラー率 |
| 15 | **分散トレーシングとサービスグラフ** | Tempo ベースのトレースを Kiali（Istio ambient）、Vizceral、promviz のトラフィックマップで可視化 |
| 16 | **カオスエンジニアリング** | Chaos Mesh による障害注入（pod-kill 等）をクラスタ内のボタンから発火し、影響範囲を Grafana で観察 |
| 17 | **継続的な負荷生成** | k6 が常時トラフィックを流し、ダッシュボードと KPI が常に生きたデータを表示 |
| 18 | **GitHub Actions による自動デプロイ** | push → build → LKE デプロイ → Akamai Functions デプロイを完全自動化 |
| 19 | **認証付き商品管理画面** | 商品の追加・編集・削除・在庫管理をブラウザから実施 |
| 20 | **独自ドメイン + HTTPS** | `tserof.net` ゾーンを Linode DNS で管理。Let's Encrypt の TLS をストア / Grafana 両方の NodeBalancer で直接終端 |

---

## アーキテクチャ

![全体アーキテクチャ](docs/architecture.svg)

<details>
<summary>詳細データフロー図 (Mermaid)</summary>

```mermaid
graph TB
    subgraph Client["クライアント（ブラウザ）"]
        USER([User])
    end

    subgraph AkamaiEdge["Akamai Functions（Fermyon Spin）"]
        INTRO["product-intro-service<br/>✨ AI 商品説明"]
        REC["recommendation-service<br/>🤖 AI レコメンド"]
        ASSIST["shopping-assistant-service<br/>💬 AI チャット"]
    end

    NB["Linode NodeBalancer<br/>🔒 TLS 終端 (Let's Encrypt)"]

    ZUPLO["Zuplo AI Gateway<br/>🛡️ Firewall for AI<br/>(PII / カード番号 / 過大入力を遮断)"]

    subgraph LKE["LKE クラスタ（東京 / jp-tyo-3 / id 610031）"]
        direction TB
        FE["frontend (Go)<br/>/, /cart, /orders, /admin/*"]
        PC["productcatalog<br/>(Go, MongoDB バックエンド)"]
        CART["cartservice (Go)"]
        CHK["checkoutservice<br/>(Go, PG に注文を永続化)"]
        CUR["currencyservice<br/>(Node.js, ライブ為替)"]
        REC2["recommendationservice (Python)"]
        SHIP["shippingservice (Go)"]
        PAY["paymentservice (Node.js)"]
        EMAIL["emailservice (Python)"]
        AD["adservice (Java)"]
        MONGO[("MongoDB<br/>StatefulSet + PVC<br/>商品カタログ")]
    end

    subgraph LinodeManaged["Linode マネージドサービス"]
        PG[("Managed PostgreSQL<br/>orders / order_items")]
        VALKEY[("Managed Valkey<br/>🔒 TLS<br/>カート + sales:ranking:units")]
        OBJ[("Object Storage<br/>akamai-boutique-img")]
    end

    subgraph GPU["GPU サーバー（Linode VM）"]
        GEMMA["llama-cpp-python<br/>Gemma 4 26B<br/>/v1/chat/completions + /metrics"]
    end

    subgraph Monitoring["クラスタ内 可観測性"]
        OTEL["OTel Collector"]
        PROM["Prometheus"]
        LOKI["Loki"]
        TEMPO["Tempo"]
        GRAF["Grafana"]
        ACLP["aclp-collector<br/>(Cloud Pulse ブリッジ)"]
        PGEXP["postgres_exporter<br/>(注文 KPI クエリ)"]
    end

    USER -->|"HTTPS<br/>aka-store.tserof.net"| NB
    NB --> FE
    USER -.->|非同期 fetch| INTRO
    USER -.->|非同期 fetch| REC
    USER -.->|チャット| ASSIST
    INTRO --> GEMMA
    REC --> GEMMA
    ASSIST --> ZUPLO
    ZUPLO --> GEMMA
    FE --> PC & CART & CHK & CUR & REC2 & SHIP & AD
    FE -->|read| PG
    FE -->|画像| OBJ
    FE -->|"ZREVRANGE（ランキング）"| VALKEY
    CHK --> PAY & EMAIL & SHIP & CART & CUR & PC
    CHK -->|write| PG
    CHK -->|"ZINCRBY（ランキング）"| VALKEY
    PC --> MONGO
    CART -->|"カート（TLS）"| VALKEY
    ACLP -->|Linode API| PG
    PGEXP -->|SQL クエリ| PG
    PROM -->|scrape| OTEL & ACLP & GEMMA & PGEXP
    GRAF --> PROM & LOKI & TEMPO
    USER -.->|"HTTPS<br/>grafana.tserof.net"| GRAF
```

</details>

---

## 技術スタック

| レイヤ | 技術 | 役割 |
|--------|------|------|
| **インフラ** | Linode Kubernetes Engine (LKE) | 東京の Kubernetes クラスタ（`jp-tyo-3`, id `610031`） |
| **エッジ AI** | Akamai Functions (Fermyon Spin v3.6.3) | TypeScript Wasm をエッジで実行 |
| **AI ゲートウェイ** | Zuplo (Firewall for AI) | チャットのプロンプトを LLM 到達前に検査（PII / カード番号 / 過大入力を遮断） |
| **AI モデル** | Gemma 4 26B (MoE, アクティブ 4B) / llama-cpp-python | GPU サーバー上のオープンソース LLM、OpenAI 互換 API |
| **DNS / TLS** | Linode DNS (`tserof.net`) + Let's Encrypt | A レコードは `linode-cli` で管理。TLS は NodeBalancer で終端（`linode-loadbalancer-port-443` CCM アノテーション、4 SAN 共有証明書） |
| **フロントエンド** | Go + HTML テンプレート | EC サイト本体（en / ja / ko / zh） |
| **マイクロサービス** | Go / Python / Node.js / Java | カート、決済、為替、配送ほか |
| **カタログストア** | MongoDB 7.0（クラスタ内 StatefulSet + PVC） | 商品カタログ（`products.json` から自動シード） |
| **注文ストア** | Linode Managed PostgreSQL | `orders` / `order_items` テーブルに永続化 |
| **カートストア** | **Akamai Managed Valkey**（Aiven ベース） | セッション単位のカートを TLS（`rediss://`）で保持。在クラスタの `redis-cart` Deployment を置き換え |
| **ランキングストア** | 同じ Valkey インスタンスの Sorted Set `sales:ranking:units` | 販売数のリーダーボード。注文ごとに `ZINCRBY` でアトミックに更新 |
| **画像ストア** | Linode Object Storage | public-read バケットから `/static/img/products/*` を配信 |
| **サービスメッシュ** | Istio **ambient**（サイドカーレス）+ waypoint、Kiali | サイドカーを注入せずに L7 テレメトリとトポロジを取得（Calico 互換性のため ambient を採用） |
| **カオスエンジニアリング** | Chaos Mesh + クラスタ内 `chaos-button` | デモ中に pod-kill 等の障害を任意のタイミングで発火 |
| **負荷生成** | k6（Locust の `loadgenerator` も残存するが停止中） | ダッシュボード / KPI が空にならないよう常時トラフィックを生成 |
| **可観測性** | Prometheus / Loki / Tempo / Grafana / Grafana Alloy / Beyla / OTel Collector | メトリクス、ログ、トレース、eBPF |
| **マネージド系メトリクス** | Akamai Cloud Pulse (`aclp-collector`) | DBaaS + NodeBalancer のメトリクスを Prometheus に橋渡し |
| **ビジネス KPI メトリクス** | `postgres_exporter`（カスタムクエリ） | 前日・当日の注文数と売上を PostgreSQL から直接クエリし Prometheus ゲージとして公開 |
| **LLM メトリクス** | prometheus-fastapi-instrumentator + カスタム middleware | モデル別 token / レイテンシ / エラー率 |
| **CI/CD** | GitHub Actions + セルフホスト Runner | push → build → デプロイの自動化 |
| **コンテナレジストリ** | GitHub Container Registry (ghcr.io) | Docker イメージの保管先 |

---

## データ永続化

4 つのドメインを 3 つのストアで永続化しています。いずれも Kubernetes マニフェスト
+ GitHub Secrets で構成され、毎回の push で CI から自動デプロイされます。

| ドメイン | 場所 | スキーマ / 形式 |
|----------|------|---------------|
| **商品カタログ** | クラスタ内 MongoDB 7.0 StatefulSet（`mongodb-0`, 10 Gi PVC） | 1 商品 1 ドキュメント（`_id` = SKU、多言語 `name` / `description`、`price`、`categories`、`picture`、`stock`）。初回起動時に `src/productcatalogservice/products.json` から自動シード。 |
| **注文** | Linode Managed PostgreSQL（`postgresql v18`, インスタンス `481207`） | `orders(order_id UUID, session_id, email, shipping_*, total_*, created_at)` + `order_items(order_id, product_id, quantity, unit_price_*)`。毎回の CI で one-shot な `orders-schema-migrate` Job が適用。 |
| **カート** | **Akamai Managed Valkey**（`Valkey9`, インスタンス `501007`） | 一時的、セッション単位。TLS + ユーザー名/パスワード認証で接続。 |
| **売上ランキング** | 同じ Valkey インスタンスの Sorted Set `sales:ranking:units` | `member` = 商品 ID、`score` = 累計販売数。`ZINCRBY` で書き込み、`ZREVRANGE ... WITHSCORES` で読み出し。 |

フロントエンドが公開するエンドポイント：

- `GET /orders` — 現在のセッションの注文履歴（cookie 紐付け）
- `GET /admin/orders` — 運用者向けの最新注文一覧（Basic 認証）
- `GET /ranking` — 売れ筋ランキング（TOP 20）。上位 8 件はホーム画面右側の
  スティッキーなサイドバーにも表示

注文の永続化もランキング更新も **ノンブロッキング**: PostgreSQL や Valkey に
到達不能な場合でもチェックアウト自体は完了し、warn ログを出すだけです。
`REDIS_URL` が未設定の場合はランキングが自動的に無効化され、Valkey 導入前と
まったく同じ挙動になります。

### Managed Valkey への接続（TLS の注意点）

クライアントを実装する前に知っておくべき点が 2 つあります。いずれも実際に
デバッグ時間を要した箇所です。

- **public エンドポイントは公的に信頼された証明書を提示します**
  （`CN=*.g2a.akamaidb.net`、Let's Encrypt 発行）。したがって
  **システムのルートストアだけで検証できます**。
  `GET /v4/databases/valkey/instances/{id}/ssl` が返す CA は*別物*の私設
  「Project CA」で、これを pin すると全接続が
  `x509: certificate signed by unknown authority` で失敗します。
- その `ca_certificate` フィールドは **base64 が二重に掛かっています**。
  1 回 base64 デコードしないと PEM として解釈できません（.NET では
  `ASN1 corrupted data` が発生）。
- Go のサービスは `FROM scratch` でビルドしているため **CA バンドルが一切
  入っていません**。Dockerfile でビルダーステージから
  `/etc/ssl/certs/ca-certificates.crt` をコピーしています。これが無いと
  上記に関わらずルート証明書ゼロで TLS が失敗します。

---

## 可観測性

`monitoring` 名前空間にセルフマネージドな可観測性スタックを置き、
Linode マネージドサービスについては Akamai Cloud Pulse から取り込みます。

| コンポーネント | 役割 |
|---------------|------|
| **Prometheus** | OTel Collector / `aclp-collector` / `postgres_exporter` / LLM `/metrics` を scrape |
| **Loki** | コンテナログ（Grafana Alloy 経由） |
| **Tempo** | マイクロサービスと Spin Function の分散トレース |
| **Grafana** | 4 ダッシュボード：*Home*（`akamai-home`）、*Operations*（`akamai-microservices`）、*Infrastructure & LLM*（`akamai-infrastructure`）、*Service Graph*（`service-graph`） |
| **Kiali** | Istio ambient のトポロジと L7 ゴールデンシグナル |
| **Vizceral / promviz** | 2 種類のトラフィックマップ。どちらも Tempo の service graph が唯一のデータソース |
| **Chaos Mesh** | 障害注入（pod-kill 等）。クラスタ内の `chaos-button` からライブで発火可能 |
| **`aclp-collector`** | Akamai 配布の OTel ディストリビューション。Cloud Pulse → Prometheus へ `dbaas` / `nodebalancer` メトリクスを橋渡し |
| **`postgres_exporter`** | `orders` テーブルに SQL で直接クエリ。`orders_daily_*`（前日）と `orders_today_*`（当日）ゲージを Grafana のビジネス KPI パネルに提供 |
| **LLM 計装** | llama-cpp-python サーバー内に `prometheus-fastapi-instrumentator` + カスタム token カウント middleware |

**Cloud Pulse から取得するメトリクス：**

- *DBaaS (PostgreSQL, entity `481207`)*: `avg_cpu_usage`, `avg_memory_usage`, `avg_disk_usage`, `avg_read_iops`, `avg_write_iops`
- *DBaaS (Valkey, entity `501007`)*: 上記と同じ 5 指標を専用のダッシュボード行に表示。
  ただし Cloud Pulse は現時点で **Valkey 固有のメトリクスを提供していません**
  （keyspace hits/misses、evicted keys、connected clients などは無く、上記の
  汎用インフラ指標のみ）。
- *NodeBalancer*: **アカウントレベルで未開放**（2026-06-02 時点でも利用不可を確認済み。
  つまり `401` が返るのは設定ミスではなく想定どおりの挙動）。設定は
  `aclp-collector.yaml` に準備済みなので、Akamai 側が開放したら有効化するだけです。

**ビジネス KPI メトリクス（PostgreSQL から直接取得）：**

- `orders_daily_order_count` — 前日（UTC）の注文数
- `orders_daily_revenue_usd` — 前日の USD 売上
- `orders_today_order_count` — 当日（UTC）の注文数（累計）
- `orders_today_revenue_usd` — 当日の USD 売上（累計）

**LLM 側で取得するメトリクス：**

- `llm_requests_total{model, endpoint, status}`
- `llm_request_duration_seconds_bucket`（latency histogram → p50 / p95 / p99）
- `llm_prompt_tokens_total`, `llm_completion_tokens_total`, `llm_total_tokens_total`

> **Object Storage（Cloud Pulse）：** 収集可能になりました。ただし
> `linode/aclp-collector:1.5.0-docker` イメージ（無印の `1.5.0` では不可）と
> **3600 秒以上**のポーリング間隔が必要です。残る障壁はスコープで、
> `LINODE_PAT` に Object Storage の read 権限が無いとバケット一覧が `401` に
> なります。なお Object Storage / Logs の Cloud Pulse は **`jp-osa`（E1）では
> 未対応**で、`jp-tyo-3`（E3）でのみ利用できます。
>
> ⚠️ collector のイメージ更新は単純な差し替えでは済みません。`1.5.0-docker`
> 以降は config スキーマが変わり `dbaas` / `nodebalancer` にも `group_by` が
> 必須になるため、現行 config のままだと
> `invalid configuration: group_by list cannot be empty` でクラッシュします。

---

## デモ環境へのアクセス

| URL | 説明 |
|-----|------|
| `https://aka-store.tserof.net/` | EC サイト（公開。`https://tserof.net/`・`https://www.tserof.net/` でも可） |
| `https://aka-store.tserof.net/ranking` | Valkey Sorted Set によるライブ売れ筋ランキング |
| `https://aka-store.tserof.net/orders` | セッション別の注文履歴 |
| `https://aka-store.tserof.net/admin/inventory` | 商品管理画面（Basic 認証） |
| `https://aka-store.tserof.net/admin/orders` | 注文一覧（Basic 認証） |
| `https://grafana.tserof.net/` | Grafana ダッシュボード |

> DNS は Linode DNS（`tserof.net` ゾーン）でホストし、TLS は Linode NodeBalancer で
> 共有の Let's Encrypt 証明書（SAN: `tserof.net` / `www` / `aka-store` / `grafana`）
> により終端。証明書の更新は `deploy-tls-tserof-tokyo.yml` ワークフローで実施。
> NodeBalancer の IP への素の HTTP アクセスもフォールバックとして利用可。
>
> ⚠️ 証明書は**手動更新**で、有効期限は **2026-10-08** です。Secret の中身だけを
> 更新しても Linode CCM は反映しません（`Service` のアノテーションが変わるまで
> 再同期しないため、ワークフロー側でアノテーションを強制的に更新しています）。
> DNS レコードの操作は CI の `LINODE_PAT` に Domains スコープが無いため、
> ローカルの `linode-cli` で実施します。

**管理画面の認証情報**

| 項目 | 値 |
|------|-----|
| ユーザー名 | `admin` |
| パスワード | `••••••••`（ここには記載しません） |

> デプロイ時に環境変数 `ADMIN_USER` / `ADMIN_PASSWORD`（Kubernetes Secret
> `frontend-admin-secret` 由来、Step 8 参照）で設定します。現在のパスワードは
> デモ管理者に確認してください。

---

## セットアップ手順

### 前提

| ツール | バージョン | 用途 |
|--------|----------|------|
| kubectl | v1.28+ | Kubernetes 操作 |
| Spin CLI | v3.6.3 | Akamai Functions の build & deploy |
| Spin aka プラグイン | v0.7.0 | Akamai Functions の認証 & デプロイ |
| Node.js | v20+ | Spin TypeScript アプリのビルド |
| AWS CLI | 最新 | Linode Object Storage（S3 互換）へ商品画像をアップロード |
| Linode CLI（任意） | 最新 | LKE クラスタの作成 |

---

### Step 1 — LKE クラスタを作成

Akamai Cloud Console または Linode CLI から作成。

```bash
linode-cli lke cluster-create \
  --label akamai-demo \
  --region jp-tyo-3 \
  --k8s_version "$(linode-cli lke versions-list --text --no-headers | head -1)" \
  --node_pools.type g6-standard-4 \
  --node_pools.count 3
```

> `jp-tyo-3` である点が重要です。Object Storage / Logs の Cloud Pulse は
> `jp-osa`（E1）では未対応で、`jp-tyo-3` のような E3 リージョンが必要です。

kubeconfig をダウンロードして接続確認：

```bash
linode-cli lke kubeconfig-view <cluster-id> --text | base64 -d > ~/.kube/config
kubectl get nodes   # 全ノードが Ready になっていれば OK
```

---

### Step 2 — Linode マネージドサービスを準備

1. **Managed PostgreSQL**（Cloud Manager → Databases → Create）
   - エンジン: PostgreSQL v16+、リージョン: LKE と同じ
   - 作成後、*Manage Networking* で **public access** を有効化
     （LKE Enterprise でないクラスタは VPC 内 private エンドポイントに
     到達できないため、public + IP allow-list がサポートされる経路）
   - LKE ノードの public IP（または閉鎖デモなら `0.0.0.0/0`）を
     データベースのファイアウォールに追加

2. **Managed Valkey**（Cloud Manager → Databases → Create → Valkey）
   - リージョン: LKE と同じ。上記と同様に public access を有効化
   - 接続情報を控えます。TLS は**必須**なので URL は `redis://` ではなく
     `rediss://` です。用途に応じて 2 種類の書式が必要になります:
     - go-redis（frontend / checkoutservice）: `rediss://user:pass@host:port`
     - StackExchange.Redis（cartservice）: `host:port,ssl=true,user=…,password=…`
   - 実装前に[データ永続化](#データ永続化)の TLS 注意点を必ず確認してください。
     CA の扱いに 2 つの落とし穴があります。

3. **Object Storage バケット**（Cloud Manager → Object Storage → Create Bucket）
   - ラベル: `akamai-boutique-img`（任意）、リージョン: LKE と同じ
   - 当該バケットに read/write 権限を持つアクセスキーを発行

4. 同梱の商品画像をアップロード：

   ```bash
   aws configure --profile linode   # アクセスキーとシークレットを入力
   bash scripts/upload-product-images.sh
   ```

---

### Step 3 — マイクロサービスをデプロイ

```bash
git clone https://github.com/ymori-aka/akamai-microservices-demo.git
cd akamai-microservices-demo

# 全マイクロサービスを一括デプロイ
kubectl apply -f kubernetes-manifests/

# プラットフォームバッジを Akamai に設定
kubectl set env deployment/frontend ENV_PLATFORM=akamai

# 全 Pod が Running になるまで待機（3〜5 分）
kubectl get pods -w
```

商品カタログは **自己シード型**：初回起動時に `productcatalogservice` が
イメージに同梱された `src/productcatalogservice/products.json` を読み、
collection が空なら MongoDB に流し込みます。2 回目以降は MongoDB だけを参照。

---

### Step 4 — 管理サーバー（GitHub Actions セルフホスト Runner）を構築

CI/CD のために、LKE クラスタにアクセスできる管理サーバー（Ubuntu 24.04 推奨）が必要です。

**GitHub Actions Runner のインストール**

```bash
# GitHub リポジトリの Settings → Actions → Runners → New self-hosted runner を開く
# 表示されるコマンドをコピー＆ペースト（トークンはその場で生成される）
mkdir -p ~/actions-runner && cd ~/actions-runner
curl -o actions-runner-linux-x64-2.321.0.tar.gz -L \
  https://github.com/actions/runner/releases/download/v2.321.0/actions-runner-linux-x64-2.321.0.tar.gz
tar xzf ./actions-runner-linux-x64-2.321.0.tar.gz
./config.sh --url https://github.com/ymori-aka/akamai-microservices-demo --token <TOKEN>
sudo ./svc.sh install && sudo ./svc.sh start
```

**管理サーバーに必要な追加ツール**

```bash
# kubectl（~/.kube/config に kubeconfig を配置）
curl -LO "https://dl.k8s.io/release/$(curl -sL https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# Spin CLI + aka プラグイン
curl -fsSL https://developer.fermyon.com/downloads/install.sh | bash
sudo mv spin /usr/local/bin/
spin plugins install aka

# Node.js v20
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt-get install -y nodejs
```

---

### Step 5 — Akamai Functions に Spin アプリをデプロイ

```bash
# 初回のみ認証
spin aka login

for svc in recommendation-service product-intro-service shopping-assistant-service; do
  (cd src/spin-functions/$svc \
    && npm install && npm run build \
    && spin aka app link --app-name $svc \
    && spin aka app deploy --no-confirm)
done
```

デプロイ後、`src/frontend/templates/product.html` のエンドポイント URL と、
`frontend.yaml` の `SHOPPING_ASSISTANT_SERVICE_ADDR` env を更新：

```text
https://<uuid-intro>.fwf.app/intro?product_id=...
https://<uuid-rec>.fwf.app/recommendations?product_id=...
https://<uuid-assistant>.fwf.app   # frontend Deployment の env に設定
```

---

### Step 6 — GitHub Secrets を設定

リポジトリの **Settings → Secrets and variables → Actions** から登録：

| Secret | 説明 |
|--------|------|
| `GHCR_TOKEN` | `write:packages` + `read:packages` 権限を持つ GitHub PAT（CI が ghcr.io へ push するほか、デプロイの度にクラスタの `ghcr-secret` がこの値から再生成される） |
| `KUBECONFIG_DATA` | セルフホスト Runner で使う kubeconfig（Base64 エンコード） |
| `SPIN_AKA_ACCESS_TOKEN` | Akamai Functions のデプロイトークン（`spin aka login --token …`） |
| `ORDER_DB_DSN` | Postgres 接続文字列：`postgres://akmadmin:<password>@<public-host>:23630/defaultdb?sslmode=require` |
| `VALKEY_URL` | frontend / checkoutservice 用の go-redis 形式：`rediss://<user>:<password>@<host>:<port>` |
| `VALKEY_ADDR` | cartservice 用の StackExchange.Redis 形式：`<host>:<port>,ssl=true,user=…,password=…` |
| `VALKEY_CA_CERT` | CA 証明書（cartservice が in-process でチェーン検証する場合のみ必要。上記 TLS 注意点を参照） |
| `LINODE_PAT` | **Monitor: Read Only** と **Databases: Read Only** スコープを持つ Linode Personal Access Token（`aclp-collector` が使用）。Object Storage の Cloud Pulse メトリクスも取りたい場合は **Object Storage: Read Only** を追加。 |

> ⚠️ **障害原因として最も多いのがトークンの失効です。** 3 種類の認証情報が
> それぞれ別のスケジュールで失効し、いずれも「トークンが原因」とは
> 分かりにくい形で症状が出ます。
>
> | Secret | 失効時の症状 |
> |--------|-------------|
> | `GHCR_TOKEN` | `ghcr-secret` が**全イメージ**に対して `DENIED` を返すようになる。稼働中の Pod はノード上のキャッシュで生き残るため、最初に犠牲になるのは「たまたま再スケジュールされた Pod」。それが `currencyservice` だとストア全体が **HTTP 500**（`could not retrieve currencies: no healthy upstream`）になる。 |
> | `LINODE_PAT` | `aclp-collector` が `401` になり、Cloud Pulse のメトリクスが無言で停止する。 |
> | `SPIN_AKA_ACCESS_TOKEN` | Akamai Functions のデプロイが失敗。仕様上**最長 90 日**なので `spin aka auth token create --expiration-days 90` で定期ローテートが必要。 |

---

### Step 7 — LLM サーバーをセットアップ（計装済み）

LLM は別の GPU VM 上で systemd サービスとして動作させます。Prometheus
メトリクスを出すために、`llama_cpp.server` を
`prometheus-fastapi-instrumentator` + 小さなカスタム middleware で
ラップします。すぐ使えるラッパーが
[`scripts/llm_server_instrumented.py`](scripts/llm_server_instrumented.py)
にあります（LLM VM 上の `/root/llm_server_instrumented.py` に手動で配置）。

systemd unit の `ExecStart` を、直接 `python -m llama_cpp.server` を
呼ぶ代わりにこのラッパー経由に切り替えます。ラッパーは以下を提供：

- `/metrics` — 標準 HTTP メトリクス + `llm_*` の token / latency カウンタ
- それ以外のルートは変更なし（`/v1/chat/completions`, `/v1/models`, …）

LKE ノードの public IP からの TCP/8000 inbound を VM の Cloud Firewall
で許可することで、Prometheus が scrape できます。

---

### Step 8 — 任意の環境変数（frontend）

```bash
kubectl set env deployment/frontend \
  ENV_PLATFORM=akamai \         # プラットフォームバッジ（akamai / gcp / aws / azure）
  ADMIN_USER=admin \            # 管理画面 Basic 認証ユーザー名
  ADMIN_PASSWORD='<パスワード>' \  # 管理画面 Basic 認証パスワード
  ENABLE_ASSISTANT=true \       # AI ショッピングアシスタントを有効化
  IMAGE_BASE_URL=https://akamai-boutique-img.jp-tyo-1.linodeobjects.com
```

---

## CI/CD パイプライン

`main` への push で並列ビルドジョブとデプロイジョブが走ります：

```
push to main
    │
    ├─► Build Frontend Image            (GitHub-hosted)
    ├─► Build Currency Service Image    (GitHub-hosted)
    ├─► Build Product Catalog Image     (GitHub-hosted)
    ├─► Build Checkout Service Image    (GitHub-hosted)
    │           │  全て ghcr.io/ymori-aka/<svc>:sha-XXXXXXX に push
    │           ▼
    ├─► Deploy to LKE                   (セルフホスト Runner)
    │      ├─ kubectl apply -f kubernetes-manifests/...
    │      ├─ frontend / checkoutservice / productcatalogservice をロールアウト
    │      ├─ Secret orders-db-credentials を作成（ORDER_DB_DSN から）
    │      ├─ orders-schema-migrate Job を実行
    │      ├─ MongoDB StatefulSet をデプロイ（不在なら）
    │      ├─ 可観測性スタック + aclp-collector をデプロイ（LINODE_PAT が設定済みなら）
    │      └─ 管理者専用商品（AKMT028, AKMT029）をシード
    │
    └─► Deploy Spin Apps to Akamai Functions
           recommendation / product-intro / shopping-assistant
```

### どのワークフローがどのクラスタを対象にするか

| ワークフロー | 対象 |
|-------------|------|
| `deploy.yml` | `main` への push で起動。デプロイジョブはセルフホスト Runner の**デフォルト kubeconfig** を使うため、以前は削除済みの大阪クラスタを指していました。**`main` への push だけでライブデモが更新されると考える前に、現在どこを向いているか必ず確認してください。** |
| `deploy-tokyo.yml` | Linode API から東京（`610031`）の kubeconfig を明示的に取得するため、Runner のデフォルトに依存せず確実です。 |
| `deploy-valkey-tokyo.yml`（`valkey` ブランチ） | `valkey-credentials` / `valkey-ca-cert` Secret を作成し、Valkey 用マニフェストを適用、`valkey-*` イメージタグを固定して実注文で検証します。 |

その他大量にある `fix-*` / `diag-*` / `chk-*` ワークフローは、過去の個別障害を
診断・修復するために書かれた使い捨てスクリプトです。汎用ツールではないので、
実行前に必ず中身を読んでください。

> ⚠️ **`deploy-tokyo.yml` はクラスタを Redis 構成に巻き戻します。**
> `main` のマニフェストと `:latest` イメージを適用するため、Valkey 構成の
> クラスタに対して実行すると、イメージタグが `valkey-*` から `:latest` に戻り、
> `REDIS_ADDR` が `redis-cart:6379` に戻り、削除したはずの `redis-cart`
> Deployment が復活し、`/ranking` が 404 になります。（`GHCR_TOKEN` ローテート
> からの復旧などで）実行した場合は、**直後に Valkey マニフェストの再適用と
> `valkey-*` イメージタグの再固定をセットで行ってください。**
> マニフェストの `image:` はプレースホルダなので `kubectl apply` だけでは
> タグは戻りません。

> **フィーチャーブランチのワークフローは、`main` にも存在しないと dispatch
> できません。** GitHub はデフォルトブランチで見えるワークフローしか登録しない
> ため、フィーチャーブランチにしか無いワークフローは
> `gh workflow run … --ref <branch>` が 404 になります。ここでの運用パターンは
> 「一時的に `main` へコミット → `--ref <branch>` で dispatch → `main` から削除」です。

---

## 商品管理画面の使い方

`http://<FRONTEND_IP>/admin/inventory` にアクセス（Basic 認証）。

| 操作 | やり方 |
|------|--------|
| 商品名 / 説明を編集 | セル内を直接編集 → "Save All Changes" |
| 価格 / 在庫を変更 | 数値フィールドを編集 → 保存 |
| 商品画像を差し替え | サムネイルをクリック → ファイル選択 → 保存 |
| 商品を非表示にする | "Hide" チェックボックスをオン → 保存 |
| 商品を削除 | 🗑 をクリック → ダイアログで確認 |
| 新しい商品を追加 | 下部のフォームに入力 → 画像をアップロード → "Add" |

`http://<FRONTEND_IP>/admin/orders`（同じ Basic 認証）では、全セッション
横断で直近 200 件の注文を一覧表示します。配送先・合計金額・明細
（`orders` / `order_items` から resolve）が見られます。

> **画像ストレージ**: `/static/img/products/` で始まる picture URL は、
> レンダリング時に `IMAGE_BASE_URL`（Linode Object Storage）に書き換えられます。
> 管理画面からアップロードした画像は現状 Pod のローカルファイルシステムにのみ
> 保存される ── 永続化したい場合はバケットへアップロード
> （`scripts/upload-product-images.sh`）して、対応するエントリを commit
> してください。

---

## リポジトリ構成

```
.
├── .github/workflows/
│   └── deploy.yml                    # CI/CD パイプライン定義
├── kubernetes-manifests/
│   ├── frontend.yaml                 # ORDER_DB_DSN, REDIS_URL, IMAGE_BASE_URL
│   ├── productcatalogservice.yaml    # MongoDB バックエンド
│   ├── checkoutservice.yaml          # PG に注文を永続化 + Valkey ランキング
│   ├── cartservice.yaml              # Valkey バックエンド（valkey では redis-cart 無し）
│   ├── mongodb.yaml                  # StatefulSet + PVC + Secret
│   ├── orders-schema.sql             # PG スキーマ（Job で適用）
│   ├── orders-migrate-job.yaml       # ワンショット psql マイグレータ
│   ├── k6.yaml                       # 継続的な負荷生成
│   ├── loadgenerator.yaml            # Locust のオーバーレイ（残存・停止中）
│   ├── hpa.yaml                      # Horizontal Pod Autoscaler
│   ├── chaos/                        # Chaos Mesh 実験 + chaos-button
│   ├── istio/                        # mTLS + カナリア（ambient mesh）
│   └── monitoring/
│       ├── prometheus.yaml
│       ├── grafana.yaml
│       ├── grafana-dashboards.yaml   # Home + Operations + Infra & LLM + Service Graph
│       ├── aclp-collector.yaml       # Akamai Cloud Pulse → Prometheus（PG + Valkey）
│       ├── aclp-collector-healer*.yaml # collector 停止時に再起動する CronJob
│       ├── postgres-exporter.yaml    # 注文 KPI クエリ → Prometheus ゲージ
│       ├── kiali.yaml                # Istio ambient のトポロジ UI
│       ├── vizceral.yaml / promviz.yaml  # トラフィックマップ（Tempo service graph 由来）
│       ├── tempo.yaml / loki.yaml / promtail.yaml
│       ├── otel-collector.yaml
│       └── redis-exporter.yaml
├── scripts/
│   ├── upload-product-images.sh      # ./src/.../products/ → Object Storage に同期
│   └── llm_server_instrumented.py    # llama_cpp.server + Prometheus 計装
├── src/
│   ├── frontend/                     # ★ 主に改修したサービス（Go）
│   │   ├── handlers.go               # ルーティング、ビジネスロジック、管理 API
│   │   ├── orders_db.go              # /orders & /admin/orders 用の PG read
│   │   ├── ranking.go                # ランキング読み出し（Valkey ZREVRANGE）
│   │   ├── translations.go           # ja / ko / zh 翻訳
│   │   ├── main.go                   # サーバ起動、ルート定義
│   │   ├── templates/
│   │   │   ├── header.html           # Akamai ロゴ、言語/通貨切替、/orders アイコン
│   │   │   ├── home.html             # Hot Products + Best Sellers サイドバー
│   │   │   ├── product.html          # AI 商品紹介 & AI レコメンド
│   │   │   ├── ranking.html          # /ranking 売れ筋ランキングページ
│   │   │   ├── orders.html           # 注文履歴ページ（両ビュー共通）
│   │   │   └── inventory.html        # 管理画面 UI
│   │   └── static/img/products/      # 元画像（Object Storage にミラー）
│   ├── cartservice/                  # C#。Valkey 接続と TLS は Startup.cs
│   ├── checkoutservice/
│   │   ├── orders_db.go              # 注文の PG 書き込み
│   │   └── ranking.go                # 注文ごとの Valkey ZINCRBY
│   ├── productcatalogservice/
│   │   ├── catalog_loader_mongo.go   # MongoDB ローダ & シーダ
│   │   └── products.json             # シードデータ
│   └── spin-functions/               # ★ Akamai Functions（TypeScript）
│       ├── recommendation-service/
│       ├── product-intro-service/
│       └── shopping-assistant-service/
└── README.md
```

---

## ライセンス・派生元について

本プロジェクトは [GoogleCloudPlatform/microservices-demo](https://github.com/GoogleCloudPlatform/microservices-demo) を派生元とし、**Apache License 2.0** に基づき利用しています。

**主な改変ポイント：**

- フロントエンドの Akamai リブランド（ロゴ、カラー、商品カタログ）
- 4 言語 UI（en / ja / ko / zh）、8 通貨対応（ライブ為替）
- Akamai Functions（Fermyon Spin）による AI 機能：商品説明、レコメンド、
  ショッピングアシスタントチャット
- 管理画面の追加（CRUD、Basic 認証、画像アップロード）
- 商品カタログを MongoDB に移行（クラスタ内 StatefulSet、自動シード）
- 注文を Linode Managed PostgreSQL に永続化、`/orders` / `/admin/orders`
  ページ追加
- カートストアを在クラスタ Redis から **Akamai Managed Valkey** へ TLS 接続で
  移行（`valkey` ブランチ）
- **ライブ売れ筋ランキング**を Valkey Sorted Set で実装。チェックアウト時に
  `ZINCRBY`、描画時に `ZREVRANGE`。ホーム画面と `/ranking` に表示
- 商品画像を Linode Object Storage から配信
- クラスタ内 Prometheus / Loki / Tempo / Grafana スタック
- Istio **ambient** メッシュ + Kiali、および Vizceral / promviz トラフィックマップ
- Chaos Mesh による障害注入とクラスタ内トリガーボタン
- k6 による継続的な負荷生成
- `aclp-collector` 経由で Akamai Cloud Pulse 統合（DBaaS（PostgreSQL +
  Valkey）と NodeBalancer のメトリクス）
- `postgres_exporter` カスタム SQL クエリによるビジネス KPI メトリクス
  （前日・当日の注文数と売上を Grafana ダッシュボードにリアルタイム表示）
- LLM サーバーを HTTP + token / latency の Prometheus メトリクスで計装
- GitHub Actions による LKE + Akamai Functions への自動デプロイ

Copyright 2018 Google LLC（オリジナルコード）— 詳細は [LICENSE](./LICENSE) を参照。
