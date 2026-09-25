#!/usr/bin/env bash
# prepare.sh - Deploys the WeKnora runtime on a clean Linux instance, used to build cloud image templates.
# No need to clone the entire WeKnora repository — only 4 runtime files (~100KB) are downloaded.
# Compatible with: Ubuntu / Debian / CentOS / Rocky / TencentOS and other distributions with systemd + Docker.
# Usage: sudo bash prepare.sh
# Adjustable environment variables:
# WEKNORA_REF              git ref to pull (tag / branch / commit), default main
# WEKNORA_DIR              deployment directory, default /opt/WeKnora
# WEKNORA_REPO             repository URL, default https://github.com/Tencent/WeKnora
# WEKNORA_GH_PROXY         GitHub acceleration prefix, default empty. Can be set on mainland China machines
# to https://gh-proxy.com/ or https://ghfast.top/
# (actual download URL becomes ${WEKNORA_GH_PROXY}${WEKNORA_REPO}/archive/...)
# DOCKER_INSTALL_MIRROR    Docker install package mirror source, default empty (uses get.docker.com).
# When mainland China machines can't reach overseas CDNs, set this to, e.g.:
#                              https://mirrors.tencent.com/docker-ce/linux/ubuntu
#                              https://mirrors.aliyun.com/docker-ce/linux/ubuntu
# it will instead install via apt + the official docker-ce repo mirror,
# including docker-ce / containerd.io / docker-compose-plugin,
# without accessing get.docker.com at all. Only supports apt-based distributions.
# DOCKER_REGISTRY_MIRROR   Docker Hub accelerator, default empty. Can be set on Tencent Cloud internal network
#                            https://mirror.ccs.tencentyun.com
# (will be written to /etc/docker/daemon.json and restart docker)
# PRUNE_OLD_IMAGES         whether to clean up dangling / old-version-tag images in upgrade scenarios,
# default false. When set to true, after pulling new images it runs
# `docker image prune -af`, removing images with no container references
# (including old WEKNORA_VERSION wechatopenai/weknora-* images)
# in one pass, reducing the size to be baked into the cloud image.
set -euo pipefail

WEKNORA_REF="${WEKNORA_REF:-main}"
WEKNORA_DIR="${WEKNORA_DIR:-/opt/WeKnora}"
WEKNORA_REPO="${WEKNORA_REPO:-https://github.com/Tencent/WeKnora}"
WEKNORA_GH_PROXY="${WEKNORA_GH_PROXY:-}"
DOCKER_INSTALL_MIRROR="${DOCKER_INSTALL_MIRROR:-}"
DOCKER_REGISTRY_MIRROR="${DOCKER_REGISTRY_MIRROR:-}"
PRUNE_OLD_IMAGES="${PRUNE_OLD_IMAGES:-false}"
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"

if [[ "${EUID}" -ne 0 ]]; then
  echo "[prepare] Please run with sudo or as root" >&2
  exit 1
fi

# Install the full docker-ce suite (including compose-plugin) via a mirror source apt.
# For mainland China cloud hosts where direct connections to get.docker.com get RST.
install_docker_via_apt_mirror() {
  local mirror="$1"
  if ! command -v apt-get >/dev/null 2>&1; then
    echo "[prepare] DOCKER_INSTALL_MIRROR currently only supports apt-based distributions (Ubuntu/Debian)" >&2
    return 1
  fi
  apt-get update -y
  apt-get install -y ca-certificates curl gnupg lsb-release
  install -m 0755 -d /etc/apt/keyrings
  curl -fsSL "${mirror%/}/gpg" | gpg --dearmor --yes -o /etc/apt/keyrings/docker.gpg
  chmod a+r /etc/apt/keyrings/docker.gpg
  local arch codename
  arch="$(dpkg --print-architecture)"
  codename="$(lsb_release -cs)"
  echo "deb [arch=${arch} signed-by=/etc/apt/keyrings/docker.gpg] ${mirror%/} ${codename} stable" \
    > /etc/apt/sources.list.d/docker.list
  apt-get update -y
  apt-get install -y docker-ce docker-ce-cli containerd.io \
                     docker-buildx-plugin docker-compose-plugin curl tar
}

echo "[prepare] 1/6 Installing Docker and dependencies"
if ! command -v docker >/dev/null 2>&1; then
  if [[ -n "${DOCKER_INSTALL_MIRROR}" ]]; then
    echo "[prepare]   Installing docker-ce via apt through ${DOCKER_INSTALL_MIRROR} (skipping get.docker.com)"
    install_docker_via_apt_mirror "${DOCKER_INSTALL_MIRROR}"
  else
    curl -fsSL https://get.docker.com | bash
  fi
fi
systemctl enable --now docker

if ! docker compose version >/dev/null 2>&1; then
  if command -v apt-get >/dev/null 2>&1; then
    apt-get update -y
    apt-get install -y docker-compose-plugin curl tar
  elif command -v yum >/dev/null 2>&1; then
    yum install -y docker-compose-plugin curl tar
  fi
fi

# Optional: configure a Docker Hub accelerator to resolve timeouts connecting directly to registry-1.docker.io
# (typical for: mainland China cloud hosts / restricted internal networks). Only touches daemon.json when explicitly provided by the user.
if [[ -n "${DOCKER_REGISTRY_MIRROR}" ]]; then
  echo "[prepare] 1.5/6 Configuring Docker Hub accelerator: ${DOCKER_REGISTRY_MIRROR}"
  mkdir -p /etc/docker
  # Merge into existing daemon.json via python, avoiding overwriting other user configuration
  if [[ -s /etc/docker/daemon.json ]] && command -v python3 >/dev/null 2>&1; then
    python3 - "$DOCKER_REGISTRY_MIRROR" <<'PY'
import json, sys, pathlib
p = pathlib.Path("/etc/docker/daemon.json")
mirror = sys.argv[1]
try:
    cfg = json.loads(p.read_text())
except Exception:
    cfg = {}
mirrors = cfg.get("registry-mirrors") or []
if mirror not in mirrors:
    mirrors.insert(0, mirror)
cfg["registry-mirrors"] = mirrors
p.write_text(json.dumps(cfg, indent=2) + "\n")
PY
  else
    cat >/etc/docker/daemon.json <<EOF
{
  "registry-mirrors": ["${DOCKER_REGISTRY_MIRROR}"]
}
EOF
  fi
  systemctl restart docker
  # Give the docker daemon a moment to restart, avoiding an immediate EOF from docker compose pull below
  for _ in 1 2 3 4 5; do
    docker info >/dev/null 2>&1 && break
    sleep 1
  done
fi

echo "[prepare] 2/6 Fetching WeKnora runtime files (ref=${WEKNORA_REF})"
# Only download the actually needed 4 files, without cloning the entire repository (~MB scale -> ~KB scale)
mkdir -p "${WEKNORA_DIR}/config"

tmp=$(mktemp -d)
trap 'rm -rf "${tmp}"' EXIT

tarball_url="${WEKNORA_GH_PROXY}${WEKNORA_REPO}/archive/${WEKNORA_REF}.tar.gz"
echo "[prepare]   tarball: ${tarball_url}"
curl -fsSL "${tarball_url}" -o "${tmp}/repo.tar.gz"
# Only extract the needed paths, significantly faster and saves space
tar -xzf "${tmp}/repo.tar.gz" -C "${tmp}" \
  --wildcards \
  '*/docker-compose.yml' \
  '*/.env.example' \
  '*/config/config.yaml'
src=$(find "${tmp}" -maxdepth 1 -mindepth 1 -type d -name 'WeKnora-*' | head -1)
if [[ -z "${src}" ]]; then
  echo "[prepare] Extraction failed, WeKnora-* directory not found" >&2
  exit 1
fi

cp    "${src}/docker-compose.yml" "${WEKNORA_DIR}/"
cp    "${src}/.env.example"       "${WEKNORA_DIR}/"
cp    "${src}/config/config.yaml" "${WEKNORA_DIR}/config/"

# Record metadata for reference during firstboot / upgrade
cat >"${WEKNORA_DIR}/.cloud-image-meta" <<EOF
WEKNORA_REF=${WEKNORA_REF}
WEKNORA_REPO=${WEKNORA_REPO}
PREPARED_AT=$(date -Iseconds)
EOF

echo "[prepare] 3/6 Preparing .env (default values, firstboot will replace with random keys)"
cd "${WEKNORA_DIR}"
[[ -f .env ]] || cp .env.example .env
sed -i 's/^GIN_MODE=.*/GIN_MODE=release/' .env || true

# Align WEKNORA_VERSION with WEKNORA_REF so docker compose pulls the
# image tag matching the ref. Unconditional override, to avoid .env retaining a stale version number left by a previous prepare.
# Tag naming convention for wechatopenai/weknora-* on Docker Hub:
#   - floating tag: main (always points to the latest build)
#   - pinned release tag: v prefix + semver (e.g. v0.7.2, v0.5.2)
# So we don't strip the v or map it to latest here.
WEKNORA_VERSION_VAL="${WEKNORA_REF}"
if grep -qE '^WEKNORA_VERSION=' .env; then
  sed -i "s|^WEKNORA_VERSION=.*|WEKNORA_VERSION=${WEKNORA_VERSION_VAL}|" .env
else
  echo "WEKNORA_VERSION=${WEKNORA_VERSION_VAL}" >>.env
fi
echo "[prepare]   -> WEKNORA_VERSION=${WEKNORA_VERSION_VAL}"

echo "[prepare] 4/6 Pull and start the default 5 persistent containers (frontend/app/docreader/postgres/redis)"
docker compose pull
docker compose up -d

# Pre-pull the sandbox image (Agent Skills runtime is run on-demand by app via docker run, not persistent)
# Without pre-pulling, the user's first Skill run would get stuck downloading
echo "[prepare] 4.5/6 Pre-pull sandbox image (for Agent Skills, not persistent)"
docker compose --profile full pull sandbox || true

# Other vector stores / observability components (qdrant, milvus, weaviate, doris, neo4j, langfuse-*, minio, dex)
# Not pre-pulled, saving 5-15GB. If the user needs to enable them:
#   cd /opt/WeKnora && docker compose --profile <name> up -d

# Upgrade scenario: clean up old-version-tag wechatopenai/weknora-* images.
# Off by default, to preserve a rollback path; enable explicitly before building an image to reduce size.
#
# Note: do not use `docker image prune -af`!
# The sandbox image is only pulled in compose, not brought up (Agent Skills runs it on-demand via app's docker run),
# no container references it, so a `prune -a` would delete the current version's sandbox along with everything else,
# defeating the purpose of prepare.sh step 4.5's sandbox pre-pull.
# Here we compare by exact tag, only deleting images under the wechatopenai/weknora-* repo whose tag isn't the current
# WEKNORA_VERSION; infrastructure images (paradedb / redis) are left untouched.
if [[ "${PRUNE_OLD_IMAGES,,}" == "true" || "${PRUNE_OLD_IMAGES}" == "1" ]]; then
  echo "[prepare] 4.6/6 Clean up old-version images under wechatopenai/weknora-* (PRUNE_OLD_IMAGES=true, keep=${WEKNORA_VERSION_VAL})"
  docker image ls --format '{{.Repository}}:{{.Tag}}' \
    | grep -E '^wechatopenai/weknora-' \
    | grep -vE ":${WEKNORA_VERSION_VAL}\$" \
    | xargs -r docker rmi -f 2>/dev/null || true
fi

echo "[prepare] 5/6 Install systemd unit"
# Detect the docker binary path, as different distros may place it in /usr/bin or /usr/local/bin
DOCKER_BIN="$(command -v docker)"
if [[ -z "${DOCKER_BIN}" ]]; then
  echo "[prepare] docker binary not found" >&2
  exit 1
fi
echo "[prepare]   docker binary: ${DOCKER_BIN}"

install -m 0644 "${SCRIPT_DIR}/systemd/weknora.service"           /etc/systemd/system/weknora.service
install -m 0644 "${SCRIPT_DIR}/systemd/weknora-firstboot.service" /etc/systemd/system/weknora-firstboot.service
install -m 0755 "${SCRIPT_DIR}/firstboot.sh"                      /usr/local/sbin/weknora-firstboot.sh

# Replace the docker path template in the systemd unit with the actual path
sed -i "s|@DOCKER_BIN@|${DOCKER_BIN}|g" /etc/systemd/system/weknora.service

systemctl daemon-reload
systemctl enable weknora.service
systemctl enable weknora-firstboot.service

echo "[prepare] 6/6 Done"
echo
echo "  WeKnora runtime deployed to ${WEKNORA_DIR}"
echo "    docker-compose.yml / config/config.yaml / .env"
echo "  Version: ${WEKNORA_REF}  (see ${WEKNORA_DIR}/.cloud-image-meta)"
echo
echo "Open a browser to  http://<host-public-IP>  to verify functionality"
echo
echo "After verifying, run cleanup and build the image:"
echo "      sudo bash ${SCRIPT_DIR}/cleanup.sh"
