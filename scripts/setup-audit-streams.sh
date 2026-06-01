#!/usr/bin/env bash
# Idempotent setup of Akamai Cloud Pulse audit-log delivery streams.
#
# Creates (if missing):
#   1. an akamai_object_storage destination pointing at the
#      akamai-audit-logs bucket (created by terraform-linode/audit_logs.tf)
#   2. an `audit_logs` stream (account-level audit) -> that destination
#   3. an `lke_audit_logs` stream (LKE K8s API audit) -> that destination
#      NOTE: lke_audit_logs requires per-account enablement by Akamai
#      support (same as NodeBalancer metrics were). Until enabled, the
#      API returns "not supported for this account at this time" and the
#      script skips it with a warning.
#
# Requires:
#   LINODE_TOKEN              PAT with Read/Write Monitor
#   AUDIT_S3_ACCESS_KEY       Object Storage access key (scoped to bucket)
#   AUDIT_S3_SECRET_KEY       Object Storage secret key
#
# Usage: LINODE_TOKEN=... AUDIT_S3_ACCESS_KEY=... AUDIT_S3_SECRET_KEY=... ./setup-audit-streams.sh
set -euo pipefail

API="https://api.linode.com/v4/monitor"
BUCKET="akamai-audit-logs"
# jp-tyo-3 (E3) — the only JP region Cloud Pulse supports for Log delivery.
HOST="jp-tyo-1.linodeobjects.com"
REGION="jp-tyo-3"
DEST_LABEL="audit-logs-objstorage"

auth=(-H "Authorization: Bearer ${LINODE_TOKEN}")
json=(-H "Content-Type: application/json")

# --- destination (find or create) ---
dest_id=$(curl -s "${auth[@]}" "${API}/streams/destinations" \
  | python3 -c "import json,sys; d=json.load(sys.stdin); print(next((x['id'] for x in d['data'] if x['label']=='${DEST_LABEL}'), ''))")

if [ -z "$dest_id" ]; then
  echo "Creating destination ${DEST_LABEL}..."
  dest_id=$(curl -s -X POST "${auth[@]}" "${json[@]}" "${API}/streams/destinations" \
    -d "{\"label\":\"${DEST_LABEL}\",\"type\":\"akamai_object_storage\",\"details\":{\"host\":\"${HOST}\",\"bucket_name\":\"${BUCKET}\",\"access_key_id\":\"${AUDIT_S3_ACCESS_KEY}\",\"access_key_secret\":\"${AUDIT_S3_SECRET_KEY}\",\"region\":\"${REGION}\",\"path\":\"audit/\"}}" \
    | python3 -c "import json,sys; print(json.load(sys.stdin)['id'])")
  echo "  destination id=${dest_id}"
else
  echo "Destination ${DEST_LABEL} already exists (id=${dest_id})"
fi

# --- streams (find or create) ---
existing=$(curl -s "${auth[@]}" "${API}/streams" \
  | python3 -c "import json,sys; print(' '.join(x['type'] for x in json.load(sys.stdin)['data']))")

for stype in audit_logs lke_audit_logs; do
  if echo "$existing" | grep -qw "$stype"; then
    echo "Stream ${stype} already exists — skipping"
    continue
  fi
  echo "Creating stream ${stype}..."
  resp=$(curl -s -X POST "${auth[@]}" "${json[@]}" "${API}/streams" \
    -d "{\"label\":\"${stype}-to-objstorage\",\"type\":\"${stype}\",\"destinations\":[${dest_id}]}")
  if echo "$resp" | grep -q '"errors"'; then
    echo "  WARN: could not create ${stype}: $(echo "$resp" | python3 -c 'import json,sys;print(json.load(sys.stdin)["errors"][0]["reason"])' 2>/dev/null || echo "$resp")"
  else
    echo "  created ${stype} id=$(echo "$resp" | python3 -c 'import json,sys;print(json.load(sys.stdin)["id"])')"
  fi
done

echo "Done."
