#!/usr/bin/env bash
# Let's Encrypt 証明書の自動更新(DNS-01 / Linode DNS)。
#   - llm.tserof.net    → NodeBalancer llm-gpu-sea の HTTPS 全ポート(:8005 が無ければ作成)
#   - tserof.net 系     → K8s Secret tserof-tls(default / monitoring) → CCM が NB に反映
# 残り日数が RENEW_WITHIN_DAYS を切ったものだけ更新する。FORCE=true で強制、
# STAGING=true は LE のテスト環境で「発行できるか」だけ試し、配布はしない。
# 秘密鍵は $WORK(一時ディレクトリ)にしか置かず、終了時に必ず削除する。
#
# 必要な Secret:
#   LINODE_CERT_TOKEN  Domains: Read/Write と NodeBalancers: Read/Write を持つ Linode トークン
#   LINODE_PAT         (既存)Tokyo クラスタの kubeconfig 取得用
set -euo pipefail

RENEW_WITHIN_DAYS=${RENEW_WITHIN_DAYS:-30}
FORCE=${FORCE:-false}
STAGING=${STAGING:-false}
API=https://api.linode.com/v4
DOMAIN_ID=3490334            # tserof.net
LLM_NB_ID=2438916            # llm-gpu-sea (us-sea)
LLM_SUBNET_ID=427955         # VPC-SEA sub-1 (10.0.0.0/24)
TOKYO_CLUSTER_ID=610031

: "${LINODE_CERT_TOKEN:?LINODE_CERT_TOKEN が未設定です(Domains RW + NodeBalancers RW のトークンを Secret に登録してください)}"

WORK=$(mktemp -d)
trap 'rm -rf "$WORK"; echo "一時ディレクトリ(鍵を含む)を削除しました"' EXIT

api() { # api METHOD PATH [JSON]
  curl -fsS -X "$1" "$API$2" -H "Authorization: Bearer $LINODE_CERT_TOKEN" -H 'Content-Type: application/json' ${3:+-d "$3"}
}

days_left() { # days_left HOST PORT SNI → 残り日数(取れなければ -1)
  local end
  end=$(echo | timeout 15 openssl s_client -connect "$1:$2" -servername "$3" 2>/dev/null | openssl x509 -noout -enddate 2>/dev/null | cut -d= -f2) || true
  [ -z "$end" ] && { echo -1; return; }
  python3 -c "import datetime as d,sys;e=d.datetime.strptime(sys.argv[1],'%b %d %H:%M:%S %Y %Z');print((e-d.datetime.utcnow()).days)" "$end"
}

echo "== certbot を準備 =="
# runner に python3-venv が無い場合は --target 方式にフォールバックする
if python3 -m venv "$WORK/venv" 2>/dev/null; then
  "$WORK/venv/bin/pip" install -q --upgrade pip certbot dnspython
  PYBIN="$WORK/venv/bin/python"; CERTBOT="$WORK/venv/bin/certbot"
else
  python3 -m pip install -q --target "$WORK/pylib" certbot dnspython
  export PYTHONPATH="$WORK/pylib"
  PYBIN=python3; CERTBOT="python3 -m certbot"
fi

# DNS-01 フック: TXT を作成し、Linode の権威 NS に出るまで待つ / 後片付けで削除
cat > "$WORK/auth.sh" <<EOF
#!/usr/bin/env bash
set -e
name="_acme-challenge"
[ "\$CERTBOT_DOMAIN" != "tserof.net" ] && name="_acme-challenge.\${CERTBOT_DOMAIN%.tserof.net}"
id=\$(curl -fsS -X POST "$API/domains/$DOMAIN_ID/records" -H "Authorization: Bearer $LINODE_CERT_TOKEN" -H 'Content-Type: application/json' \
      -d "{\"type\":\"TXT\",\"name\":\"\$name\",\"target\":\"\$CERTBOT_VALIDATION\",\"ttl_sec\":30}" | python3 -c 'import json,sys;print(json.load(sys.stdin)["id"])')
echo "\$id" >> "$WORK/record-ids"
PYTHONPATH="$WORK/pylib" $PYBIN - "\$CERTBOT_DOMAIN" "\$CERTBOT_VALIDATION" <<'PY'
import sys, time, dns.resolver
fqdn, want = "_acme-challenge." + sys.argv[1], sys.argv[2]
r = dns.resolver.Resolver(configure=False)
r.nameservers = [str(a) for a in dns.resolver.resolve("ns1.linode.com", "A")]
for _ in range(60):
    try:
        if any(want in b"".join(x.strings).decode() for x in r.resolve(fqdn, "TXT")):
            sys.exit(0)
    except Exception:
        pass
    time.sleep(10)
sys.exit("TXT が権威 NS に出ませんでした: " + fqdn)
PY
EOF
cat > "$WORK/cleanup.sh" <<EOF
#!/usr/bin/env bash
[ -f "$WORK/record-ids" ] || exit 0
while read -r id; do curl -fsS -X DELETE "$API/domains/$DOMAIN_ID/records/\$id" -H "Authorization: Bearer $LINODE_CERT_TOKEN" >/dev/null || true; done < "$WORK/record-ids"
rm -f "$WORK/record-ids"
EOF
chmod +x "$WORK/auth.sh" "$WORK/cleanup.sh"

issue() { # issue NAME DOMAIN... → $WORK/conf/live/NAME/{fullchain,privkey}.pem
  local name=$1; shift
  local args=(); for d in "$@"; do args+=(-d "$d"); done
  local extra=(); [ "$STAGING" = "true" ] && extra+=(--test-cert)
  $CERTBOT certonly --manual --preferred-challenges dns \
    --manual-auth-hook "$WORK/auth.sh" --manual-cleanup-hook "$WORK/cleanup.sh" \
    --config-dir "$WORK/conf" --work-dir "$WORK/work" --logs-dir "$WORK/logs" \
    --non-interactive --agree-tos --register-unsafely-without-email \
    --cert-name "$name" "${extra[@]}" "${args[@]}"
  openssl x509 -in "$WORK/conf/live/$name/fullchain.pem" -noout -subject -enddate
}

rc=0

# ---------------------------------------------------------------- llm.tserof.net
echo "== llm.tserof.net =="
configs=$(api GET "/nodebalancers/$LLM_NB_ID/configs" | python3 -c '
import json,sys
for c in json.load(sys.stdin)["data"]:
    print(c["id"], c["port"], c["protocol"])')
need_llm=$FORCE
has8005=false
while read -r id port proto; do
  [ "$proto" = "https" ] || continue
  [ "$port" = "8005" ] && has8005=true
  d=$(days_left llm.tserof.net "$port" llm.tserof.net)
  echo "  :$port 残り ${d} 日"
  [ "$d" -lt "$RENEW_WITHIN_DAYS" ] && need_llm=true
done <<< "$configs"
[ "$has8005" = "false" ] && { echo "  :8005 (Laya) が未作成 → 作成する"; need_llm=true; }

if [ "$need_llm" = "true" ]; then
  issue llm llm.tserof.net
  if [ "$STAGING" != "true" ]; then
    body=$(python3 -c 'import json,sys;print(json.dumps({"ssl_cert":open(sys.argv[1]).read(),"ssl_key":open(sys.argv[2]).read()}))' \
           "$WORK/conf/live/llm/fullchain.pem" "$WORK/conf/live/llm/privkey.pem")
    while read -r id port proto; do
      [ "$proto" = "https" ] || continue
      api PUT "/nodebalancers/$LLM_NB_ID/configs/$id" "$body" >/dev/null && echo "  :$port 更新"
    done <<< "$configs"
    if [ "$has8005" = "false" ]; then
      create=$(python3 -c '
import json,sys
b=json.loads(sys.argv[1]); b.update({"port":8005,"protocol":"https","algorithm":"roundrobin","stickiness":"none",
  "check":"http","check_path":"/health","check_interval":15,"check_timeout":5,"check_attempts":3,"check_passive":True,
  "cipher_suite":"recommended"}); print(json.dumps(b))' "$body")
      cfg=$(api POST "/nodebalancers/$LLM_NB_ID/configs" "$create" | python3 -c 'import json,sys;print(json.load(sys.stdin)["id"])')
      api POST "/nodebalancers/$LLM_NB_ID/configs/$cfg/nodes" \
        "{\"address\":\"10.0.0.2:8010\",\"label\":\"laya-vpc\",\"mode\":\"accept\",\"weight\":50,\"subnet_id\":$LLM_SUBNET_ID}" >/dev/null
      echo "  :8005 を作成(backend 10.0.0.2:8010 = Laya)"
    fi
  fi
else
  echo "  更新不要"
fi

# ---------------------------------------------------------------- tserof.net 系
echo "== tserof.net / www / aka-store / grafana =="
d1=$(days_left tserof.net 443 tserof.net); d2=$(days_left grafana.tserof.net 443 grafana.tserof.net)
echo "  ストア 残り ${d1} 日 / Grafana 残り ${d2} 日"
if [ "$FORCE" = "true" ] || [ "$d1" -lt "$RENEW_WITHIN_DAYS" ] || [ "$d2" -lt "$RENEW_WITHIN_DAYS" ]; then
  issue tserof tserof.net www.tserof.net aka-store.tserof.net grafana.tserof.net
  if [ "$STAGING" != "true" ]; then
    : "${LINODE_PAT:?LINODE_PAT が未設定です}"
    KC="$WORK/kc.yaml"
    curl -fsS -H "Authorization: Bearer $LINODE_PAT" "$API/lke/clusters/$TOKYO_CLUSTER_ID/kubeconfig" \
      | python3 -c 'import sys,json,base64;print(base64.b64decode(json.load(sys.stdin)["kubeconfig"]).decode())' > "$KC"
    export KUBECONFIG="$KC"
    for ns in default monitoring; do
      kubectl create secret tls tserof-tls -n "$ns" \
        --cert="$WORK/conf/live/tserof/fullchain.pem" --key="$WORK/conf/live/tserof/privkey.pem" \
        --dry-run=client -o yaml | kubectl apply -f -
    done
    # Secret の中身を変えただけでは Linode CCM は NodeBalancer に反映しないので注釈で同期させる
    ts=$(date -u +%Y%m%dT%H%M%SZ)
    kubectl annotate service frontend-external tserof.net/tls-synced-at="$ts" --overwrite
    kubectl -n monitoring annotate service grafana tserof.net/tls-synced-at="$ts" --overwrite
    echo "  K8s Secret を更新し CCM に同期を指示"
  fi
else
  echo "  更新不要"
fi

# ---------------------------------------------------------------- 検証
if [ "$STAGING" != "true" ]; then
  echo "== 検証(反映に最大数分) =="
  sleep 60
  for t in "llm.tserof.net 8001" "llm.tserof.net 8004" "llm.tserof.net 8005" "tserof.net 443" "grafana.tserof.net 443"; do
    set -- $t
    ok=false
    for i in $(seq 1 12); do
      d=$(days_left "$1" "$2" "$1")
      [ "$d" -ge "$RENEW_WITHIN_DAYS" ] && { ok=true; break; }
      sleep 15
    done
    echo "  $1:$2 残り ${d} 日 $([ "$ok" = true ] && echo OK || echo '★NG')"
    [ "$ok" = true ] || rc=1
  done
fi
exit $rc
