#!/bin/sh
# Render garage.toml from the template and start the server.
set -eu

: "${GARAGE_RPC_SECRET:?GARAGE_RPC_SECRET is required (64 hex chars)}"
: "${GARAGE_ADMIN_TOKEN:?GARAGE_ADMIN_TOKEN is required}"

sed -e "s/@GARAGE_RPC_SECRET@/${GARAGE_RPC_SECRET}/g" \
  -e "s/@GARAGE_ADMIN_TOKEN@/${GARAGE_ADMIN_TOKEN}/g" \
  /etc/garage.toml.template > /etc/garage.toml

exec /usr/local/bin/garage server
