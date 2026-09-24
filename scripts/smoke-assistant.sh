#!/usr/bin/env bash
# Post-deploy smoke test for the store assistant (/bot).
#
# Why this exists: fixes to the chat kept "coming back undone". The actual
# causes were never visible from a single request against the public URL,
# because frontend-external load-balances across the stable pods AND
# frontend-canary (~1/3 of traffic), and deploy.yml re-applied canary.yaml
# with a stale image after the canary had been synced. A request that happened
# to land on a stable pod looked fine while a third of real users got Japanese
# replies to English questions, Markdown, and no routing/latency badges.
#
# So this script checks every frontend pod individually (port-forward, not the
# Service), in both UI languages, and fails if any pod is wrong.
#
# Needs KUBECONFIG pointing at the Tokyo cluster. Exit code 0 = all good.
set -u
fail=0

stable=$(kubectl get deploy frontend -o jsonpath='{.spec.template.spec.containers[0].image}')
canary=$(kubectl get deploy frontend-canary -o jsonpath='{.spec.template.spec.containers[0].image}' 2>/dev/null || true)
echo "stable image: $stable"
echo "canary image: ${canary:-(none)}"
if [ -n "$canary" ] && [ "$canary" != "$stable" ]; then
  echo "::error::frontend-canary runs a different image from stable. It receives part of the"
  echo "::error::traffic, so users would see old behaviour on some requests."
  fail=1
fi

cat > /tmp/smoke-check.py <<'PY'
import json, re, sys
want = sys.argv[1]
want_router = sys.argv[2] if len(sys.argv) > 2 else ""
raw = sys.stdin.read()
try:
    d = json.loads(raw)
except Exception:
    print("NG: not JSON: " + raw[:120]); sys.exit(1)
msg = d.get("message") or ""
meta = d.get("meta") or {}
problems, warns = [], []
if msg.startswith("[DEBUG]"):
    problems.append("DEBUG reply: " + msg[:100])
if not meta:
    problems.append("no meta (old image?)")
elif meta.get("status") != 200:
    problems.append("status=%s: %s" % (meta.get("status"), msg[:80]))
has_ja = bool(re.search(r"[ぁ-んァ-ン一-龥]", msg))
if want == "ja" and not has_ja:
    problems.append("expected Japanese reply, got: " + msg[:60])
if want == "en" and has_ja:
    problems.append("expected English reply, got: " + msg[:60])
if "**" in msg or re.search(r"^\s*(\||#{1,6} |[-*] )", msg, re.M):
    warns.append("markdown in reply")
if want_router and d.get("router") != want_router:
    problems.append("answered by router=%s, expected %s" % (d.get("router"), want_router))
if not d.get("routing"):
    warns.append("no routing (classifier timeout?)")
if problems:
    print("NG: " + " / ".join(problems + warns)); sys.exit(1)
print("OK" + (" (warn: " + ", ".join(warns) + ")" if warns else "") + " | " + msg[:50].replace("\n", " "))
PY

# Classifier radio: "qwen" is always on; the Laya apps only once their URL and
# key are in the zuplo-router-apps secret (the UI greys them out until then).
routers="qwen"
for r in laya laya-sr; do
  a=$(kubectl get secret zuplo-router-apps -o go-template="{{index .data \"$r-addr\"}}" 2>/dev/null || true)
  k=$(kubectl get secret zuplo-router-apps -o go-template="{{index .data \"$r-key\"}}" 2>/dev/null || true)
  case "$a$k" in *"no value"*|"") ;; *) [ -n "$a" ] && [ -n "$k" ] && routers="$routers $r" ;; esac
done
echo "classifiers: $routers"

port=18080
for pod in $(kubectl get pod -l app=frontend --field-selector=status.phase=Running -o name); do
  img=$(kubectl get "$pod" -o jsonpath='{.spec.containers[0].image}')
  kubectl port-forward "$pod" "$port:8080" >/dev/null 2>&1 &
  pf=$!
  for i in 1 2 3 4 5 6 7 8 9 10; do curl -s -o /dev/null "http://127.0.0.1:$port/_healthz" && break; sleep 1; done
  cases=("qwen|en|Do you have any men's polo shirts?" "qwen|ja|メンズのポロシャツはありますか？")
  for r in $routers; do
    [ "$r" = qwen ] || cases+=("$r|en|Do you have any men's polo shirts?")
  done
  for c in "${cases[@]}"; do
    router=${c%%|*}; rest=${c#*|}; lang=${rest%%|*}; q=${rest#*|}
    body=$(python3 -c 'import json,sys; print(json.dumps({"message":sys.argv[1],"history":[],"lang":sys.argv[2],"router":sys.argv[3],"nocache":True}))' "$q" "$lang" "$router")
    verdict=$(curl -s -m 120 -X POST "http://127.0.0.1:$port/bot" -H 'Content-Type: application/json' -d "$body" | python3 /tmp/smoke-check.py "$lang" "$router")
    rc=$?
    echo "${pod#pod/} [$img] router=$router UI=$lang -> $verdict"
    if [ $rc -ne 0 ]; then
      echo "::error::${pod#pod/} router=$router UI=$lang: $verdict"
      fail=1
    fi
  done
  kill "$pf" 2>/dev/null; wait "$pf" 2>/dev/null
  port=$((port + 1))
done

if [ $fail -eq 0 ]; then echo "ALL PODS OK"; else echo "SMOKE TEST FAILED"; fi
exit $fail
