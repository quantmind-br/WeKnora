#!/usr/bin/env bash
# firstboot.sh - automatically executed by weknora-firstboot.service the first time a new instance boots.
# Task: generate random keys and write to .env -> start containers -> output credentials -> mark done + self-disable.
#
# Idempotency strategy:
# First write the ${MARKER} flag "keys already generated",
# so that even if docker compose fails and interrupts the script,
# the unit's ConditionPathExists=!${MARKER} blocks it after restart, preventing a new key from being generated and overwriting .env.
# (The old key is already written to the postgres data volume; generating a new one would permanently lock the database out.)
# On failure the unit is marked failed, and the user can manually recover with docker compose up -d;
# credentials can still be found in ${ENV_FILE}.
set -euo pipefail

WEKNORA_DIR="${WEKNORA_DIR:-/opt/WeKnora}"
ENV_FILE="${WEKNORA_DIR}/.env"
ENV_TEMPLATE="${WEKNORA_DIR}/.env.example"
CRED_FILE="/root/weknora-credentials.txt"
LOG_FILE="/var/log/weknora-firstboot.log"
MARKER="${WEKNORA_DIR}/.firstboot.done"

# Open LOG_FILE early, and also copy stderr to the systemd journal for easier debugging
# (with plain exec >> LOG_FILE, a failure at the start of the script would be invisible because stderr has already been swallowed)
mkdir -p "$(dirname "${LOG_FILE}")"
exec > >(tee -a "${LOG_FILE}") 2>&1
echo "==== firstboot started at $(date -Iseconds) ===="

if [[ -f "${MARKER}" ]]; then
  echo "marker ${MARKER} exists, skip (already initialized)"
  exit 0
fi

# cleanup.sh no longer keeps .env; here we copy the template from .env.example and do the substitution instead.
# This guarantees that before firstboot, there is no .env with plaintext default passwords that would let weknora.service
# initialize the postgres data volume with the wrong password ahead of time.
if [[ ! -f "${ENV_FILE}" ]]; then
  if [[ -f "${ENV_TEMPLATE}" ]]; then
    echo "creating ${ENV_FILE} from ${ENV_TEMPLATE}"
    cp "${ENV_TEMPLATE}" "${ENV_FILE}"
  else
    echo "ERROR: neither ${ENV_FILE} nor ${ENV_TEMPLATE} found"
    exit 1
  fi
fi

DOCKER_BIN="$(command -v docker || true)"
if [[ -z "${DOCKER_BIN}" ]]; then
  echo "ERROR: docker binary not found in PATH"
  exit 1
fi

# Generate a 32-byte strong random string (for the AES-256 key, must be exactly 32 bytes)
# Use `() ... ()` to start a subshell and disable pipefail: head reads only N bytes then closes stdin,
# tr receives SIGPIPE (exit code 141), and under `set -o pipefail` the whole pipeline is judged as failed,
# triggering the top-level `set -e` to kill firstboot.sh with exit 1.
gen32() ( set +o pipefail; LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 32 )
# Generic password: 24 characters, no / + = (to avoid issues appearing in URLs / during sed substitution)
genpw() ( set +o pipefail; LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 24 )

DB_PWD=$(genpw)
REDIS_PWD=$(genpw)
JWT=$(genpw)$(genpw)
SYS_AES=$(gen32)

# Use | as the sed delimiter to avoid conflicts; only replace lines starting with KEY=
replace() {
  local key="$1" val="$2"
  if grep -qE "^${key}=" "${ENV_FILE}"; then
    sed -i "s|^${key}=.*|${key}=${val}|" "${ENV_FILE}"
  else
    echo "${key}=${val}" >>"${ENV_FILE}"
  fi
}

replace DB_PASSWORD     "${DB_PWD}"
replace REDIS_PASSWORD  "${REDIS_PWD}"
replace JWT_SECRET      "${JWT}"
replace SYSTEM_AES_KEY  "${SYS_AES}"
replace GIN_MODE        "release"

# Restore the WEKNORA_REF recorded in .cloud-image-meta during the prepare.sh stage back into .env's
# WEKNORA_VERSION, otherwise docker compose would fall back to the :latest default, causing the image version
# to mismatch the version pulled during prepare.
META_FILE="${WEKNORA_DIR}/.cloud-image-meta"
if [[ -f "${META_FILE}" ]]; then
  META_REF=$(grep -E '^WEKNORA_REF=' "${META_FILE}" | tail -1 | cut -d= -f2- || true)
  if [[ -n "${META_REF}" ]]; then
    replace WEKNORA_VERSION "${META_REF}"
    echo "restored WEKNORA_VERSION=${META_REF} from ${META_FILE}"
  fi
fi

# Critical: write the marker immediately after .env is edited.
# After this, even if docker compose up fails, a restart won't rewrite .env again,
# preventing a mismatch with the password already persisted in postgres.
umask 077
touch "${MARKER}"
chmod 0600 "${MARKER}"

echo "env updated, marker written, starting docker compose..."
cd "${WEKNORA_DIR}"
"${DOCKER_BIN}" compose up -d

# Try to get the public IP, fall back to the private IP on failure
PUB_IP=$(curl -fsS --max-time 5 https://ifconfig.me 2>/dev/null \
  || curl -fsS --max-time 5 https://api.ipify.org 2>/dev/null \
  || hostname -I | awk '{print $1}')

cat >"${CRED_FILE}" <<INFO
========================================
  WeKnora instance initialization complete
  Generated at: $(date -Iseconds)
========================================

Access URL: http://${PUB_IP}

To disable further registration after signup, edit ${ENV_FILE}:
    DISABLE_REGISTRATION=true
Then run: cd ${WEKNORA_DIR} && docker compose up -d

The following random credentials were written to ${ENV_FILE}; keep them safe (root-readable only):
  DB_PASSWORD     = ${DB_PWD}
  REDIS_PASSWORD  = ${REDIS_PWD}
  JWT_SECRET      = ${JWT}
  SYSTEM_AES_KEY  = ${SYS_AES}

Note:
  - This file only reflects the secrets at first boot; ${ENV_FILE} is authoritative afterwards.
  - Never expose infrastructure ports such as 5432 / 6379 / 9000 directly.
  - Serve externally only on 80 / 443, with a reverse proxy + HTTPS if needed.

INFO

echo "credentials written to ${CRED_FILE}"

# Only stop the unit, don't delete the unit file (otherwise the currently running oneshot might get marked failed by systemd).
# On the next restart, weknora-firstboot.service automatically skips via ConditionPathExists=!${MARKER}.
systemctl disable weknora-firstboot.service || true

echo "==== firstboot finished at $(date -Iseconds) ===="
