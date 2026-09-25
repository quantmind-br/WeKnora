#!/bin/sh
# One-shot Garage cluster setup: assign layout, import the S3 key,
# create the bucket and grant access. Skips when already initialized
# (sentinel file lives on the shared garage_data volume).
set -eu

: "${GARAGE_RPC_SECRET:?GARAGE_RPC_SECRET is required}"
: "${GARAGE_ADMIN_TOKEN:?GARAGE_ADMIN_TOKEN is required}"
: "${S3_ACCESS_KEY:?S3_ACCESS_KEY is required (GK + 24 hex chars)}"
: "${S3_SECRET_KEY:?S3_SECRET_KEY is required (64 hex chars)}"
: "${S3_BUCKET_NAME:?S3_BUCKET_NAME is required}"

GARAGE=/usr/local/bin/garage
SENTINEL=/var/lib/garage/.weknora-init-done

sed -e "s/@GARAGE_RPC_SECRET@/${GARAGE_RPC_SECRET}/g" \
  -e "s/@GARAGE_ADMIN_TOKEN@/${GARAGE_ADMIN_TOKEN}/g" \
  /etc/garage.toml.template > /etc/garage.toml

if [ -f "$SENTINEL" ]; then
  echo "garage already initialized, skipping"
  exit 0
fi

i=0
until "$GARAGE" status >/dev/null 2>&1; do
  i=$((i + 1))
  if [ "$i" -ge 90 ]; then
    echo "timed out waiting for garage server" >&2
    exit 1
  fi
  sleep 2
done

NODE_ID=$("$GARAGE" status | awk '/^[0-9a-f]{16} /{print $1; exit}')
if [ -z "$NODE_ID" ]; then
  echo "could not determine garage node id" >&2
  "$GARAGE" status >&2 || true
  exit 1
fi
echo "node id: $NODE_ID"

"$GARAGE" layout assign -z dc1 -c "${GARAGE_CAPACITY:-50G}" "$NODE_ID"
"$GARAGE" layout apply --version 1
"$GARAGE" key import --yes -n weknora "$S3_ACCESS_KEY" "$S3_SECRET_KEY"
"$GARAGE" bucket create "$S3_BUCKET_NAME"
"$GARAGE" bucket allow --read --write --owner "$S3_BUCKET_NAME" --key "$S3_ACCESS_KEY"

touch "$SENTINEL"
echo "garage initialization done"
