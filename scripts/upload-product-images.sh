#!/usr/bin/env bash
# Upload existing product images to Linode Object Storage.
#
# Approach A (URL rewrite) maps each frontend image path
#   /static/img/products/<rel>
# to the bucket URL
#   https://akamai-boutique-img.jp-tyo-1.linodeobjects.com/<rel>
#
# So we upload everything under src/frontend/static/img/products/
# with object keys equal to <rel> (i.e. *without* the static/img/products
# prefix).
#
# Requirements:
#   - AWS CLI installed (`brew install awscli`)
#   - `aws configure --profile linode` already done with the Linode
#     Object Storage access/secret keys
#   - Bucket already created and set to public-read for objects

set -euo pipefail

BUCKET="akamai-boutique-img"
REGION="jp-tyo-1"
ENDPOINT="https://${REGION}.linodeobjects.com"
PROFILE="${AWS_PROFILE:-linode}"

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SRC="${ROOT}/src/frontend/static/img/products"

if [[ ! -d "${SRC}" ]]; then
  echo "source directory not found: ${SRC}" >&2
  exit 1
fi

echo "Uploading from ${SRC} to s3://${BUCKET} (endpoint ${ENDPOINT})..."

aws s3 sync "${SRC}" "s3://${BUCKET}" \
  --endpoint-url "${ENDPOINT}" \
  --profile "${PROFILE}" \
  --acl public-read \
  --exclude ".DS_Store" \
  --no-progress

echo
echo "Done. Smoke-test a known object:"
echo "  curl -I https://${BUCKET}.${REGION}.linodeobjects.com/mug.jpg"
