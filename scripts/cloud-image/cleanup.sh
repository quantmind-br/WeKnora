#!/usr/bin/env bash
# cleanup.sh - Clean up private data before building the cloud image.
# WARNING: this script deletes SSH public keys, wipes the database and logs, and shuts down automatically at the end.
# After running it, go straight to the cloud console to "Create Image / Create Snapshot / Create AMI" — do not SSH in again.
set -euo pipefail

WEKNORA_DIR="${WEKNORA_DIR:-/opt/WeKnora}"

if [[ "${EUID}" -ne 0 ]]; then
  echo "[cleanup] Please run with sudo or as root" >&2
  exit 1
fi

read -r -p "[cleanup] This operation is irreversible, continue? Type YES to continue:" ans
if [[ "${ans}" != "YES" ]]; then
  echo "[cleanup] Cancelled"
  exit 0
fi

echo "[cleanup] 1/8 Stopping WeKnora containers"
COMPOSE_PROJECT=""
if [[ -d "${WEKNORA_DIR}" ]]; then
  cd "${WEKNORA_DIR}"
  # Prefer compose ls to get the real project name (defaults to the lowercase directory name, e.g. weknora)
  COMPOSE_PROJECT="$(docker compose ls --format json 2>/dev/null \
    | grep -oE '"Name":"[^"]+"' | head -1 | cut -d'"' -f4 || true)"
  docker compose down -v --remove-orphans || true
fi

echo "[cleanup] 2/8 Wiping WeKnora business data + first-boot marker / logs"
if [[ -d "${WEKNORA_DIR}" ]]; then
  rm -rf "${WEKNORA_DIR}/data"/* "${WEKNORA_DIR}/logs"/* 2>/dev/null || true
  # Deliberately not recreating .env here: leaving .env missing in the image so any
  # docker compose that comes up before firstboot fails for lack of .env, avoiding
  # corrupting the postgres data volume with plaintext default passwords (like postgres123!@#).
  # firstboot.sh will copy from .env.example and replace the secrets itself.
  rm -f "${WEKNORA_DIR}/.env" "${WEKNORA_DIR}/.firstboot.done"
fi
rm -f /root/weknora-credentials.txt /var/log/weknora-firstboot.log

echo "[cleanup] 3/8 Cleaning up leftover docker volumes and build cache"
# Strictly match by compose project name prefix to avoid harming other postgres/redis volumes on the same host.
if [[ -n "${COMPOSE_PROJECT}" ]]; then
  docker volume ls -q --filter "label=com.docker.compose.project=${COMPOSE_PROJECT}" \
    | xargs -r docker volume rm -f || true
fi
# Note: this only cleans "volumes / stopped containers / build cache" — images must never be cleaned.
# Previously used `docker system prune -af --volumes`, which also wiped out the
# wechatopenai/weknora-* images pre-pulled by prepare.sh, causing new instances built from the image
# to have to re-pull several GB of images from Docker Hub at firstboot, defeating the whole point of pre-installing.
docker container prune -f      || true
docker builder    prune -af    || true
# Only clean unmounted dangling volumes (business volumes are already cleaned by compose down -v at this point)
docker volume     prune -f     || true

echo "[cleanup] 4/8 Clearing system logs"
journalctl --rotate || true
journalctl --vacuum-time=1s || true
find /var/log -type f \( -name '*.log' -o -name '*.gz' -o -name '*.[0-9]' \) -print0 \
  | xargs -0 -r truncate -s 0 || true
find /var/log -type f \( -name '*.gz' -o -name '*.[0-9]' \) -print0 \
  | xargs -0 -r rm -f || true

echo "[cleanup] 5/8 Cleaning up SSH history and authorized keys (you will not be able to SSH in after this runs)"
rm -f /root/.ssh/authorized_keys /root/.ssh/known_hosts /root/.bash_history
for d in /home/*; do
  [[ -d "$d" ]] || continue
  rm -f "$d/.ssh/authorized_keys" "$d/.ssh/known_hosts" "$d/.bash_history"
done
find / -xdev -type f \( -name 'id_rsa*' -o -name '*.pem' -o -name '*.key' \) \
  -not -path '/etc/ssl/*' -not -path '/usr/*' -not -path '/var/lib/docker/*' 2>/dev/null \
  | tee /tmp/cleanup-secrets-found.txt || true
echo "[cleanup]   ↑ the above are suspected leftover key files; double-check manually if needed"

echo "[cleanup] 6/8 Resetting cloud-init / machine-id (so the new instance gets a fresh ID)"
cloud-init clean --logs --seed 2>/dev/null || true
truncate -s 0 /etc/machine-id || true
rm -f /var/lib/dbus/machine-id || true

echo "[cleanup] 7/8 Cleaning up apt / tmp"
if command -v apt-get >/dev/null 2>&1; then
  apt-get clean
  rm -rf /var/lib/apt/lists/*
fi
rm -rf /tmp/* /var/tmp/* /root/.cache /home/*/.cache 2>/dev/null || true

echo "[cleanup] 8/8 Syncing disk and shutting down"
history -c || true
sync
echo
echo "Shutting down now. Once it's done, go to the cloud console to "Create Image / Create Snapshot / Create AMI"."
echo
sleep 3
poweroff
