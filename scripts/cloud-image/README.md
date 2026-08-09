# WeKnora Cloud Image Packaging Scripts (Cloud-Agnostic)

> **This document is for users who want to package WeKnora into a cloud image (AMI / custom image / snapshot) to distribute to others.**
> **If you just want to use WeKnora yourself, go straight to the main repo [README](../../README.md) — `docker compose up -d` is all you need.**

## What these scripts do

They help you turn any "Linux instance capable of running Docker" into a **distributable cloud image template**:

- After someone creates a new instance based on this image, **on first boot it will automatically**:
  - Generate brand-new random secrets (DB / Redis / JWT / AES)
  - Start all of WeKnora's default containers
  - Write the generated credentials to `/root/weknora-credentials.txt`
  - Self-delete the one-time init script
- This achieves "**ready to use on boot, zero secret leakage, unique keys per instance**"

Supported platforms (any systemd + Docker Linux works):

- Tencent Cloud Lighthouse / CVM
- AWS EC2 AMI
- Alibaba Cloud ECS custom image
- Volcengine / Huawei Cloud / Vultr Snapshot
- Local KVM / Proxmox templates

For platform-specific steps on "building the image / sharing it / listing it," see the corresponding docs under [`docs/cloud-image/`](../../docs/cloud-image/).

---

## Directory structure

```
scripts/cloud-image/
├── README.md                # This document
├── prepare.sh               # Step 1: install Docker + pull runtime files + install firstboot
├── cleanup.sh               # Step 2: clean up before making the image (locks SSH after running)
├── firstboot.sh             # Runs automatically on new instance's first boot (invisible to the user)
└── systemd/
    ├── weknora.service           # Auto-starts docker compose on boot
    └── weknora-firstboot.service # First-boot init (self-deletes after running)
```

## No need to clone the whole WeKnora repo

All WeKnora containers are pulled from Docker Hub (`wechatopenai/weknora-*`) — no Go / Python / frontend source needs to be brought onto the host.

The only things `docker-compose.yml` actually mounts from the host into the containers are:

```
- ./config/config.yaml      (single file)
- ./skills/preloaded/       (directory)
```

So the runtime files needed in the image total **less than 100KB**:

| File | Size | Purpose |
|---|---|---|
| `docker-compose.yml` | 12K | Container orchestration |
| `.env` | 12K | Environment variables |
| `config/config.yaml` | 8K | Backend business config |
| `skills/preloaded/` | 56K | Preloaded agent skills |

`prepare.sh` uses `curl + tar` to download just these 4 items — no `git clone`.

## Which containers start in the image

WeKnora's `docker-compose.yml` gates most services behind **profiles**; this image only starts the 5 core services by default.

**Started by default (5 always-on containers, auto-start on boot):**

| Container | Role |
|---|---|
| `frontend` | Vue UI / NGINX reverse proxy |
| `app` | WeKnora Go backend |
| `docreader` | Python document parsing (gRPC) |
| `postgres` (ParadeDB) | Primary DB + pgvector vector search + BM25 |
| `redis` | Streaming output / caching / async queue |

> ParadeDB bundles pgvector, so no separate vector database is needed by default.

**Pre-pulled extra, but not always-on:**

- `sandbox` image: Agent Skills are run on-demand by `app` via `docker run`. Pulling it in advance avoids new instances stalling on a download the first time a Skill runs.

**Profile-gated, not preinstalled (users pull these themselves as needed):**

| profile | Purpose |
|---|---|
| `minio` | Object storage as an alternative to local files |
| `qdrant` / `milvus` / `weaviate` / `doris` | Alternatives to pgvector |
| `neo4j` | GraphRAG knowledge graph |
| `langfuse` | Self-hosted Langfuse observability platform |
| `dex` | OIDC login |
| `odl-hybrid` | OpenDataLoader Docling hybrid (large, no prebuilt image, requires `--build`) |

How to enable:

```bash
cd /opt/WeKnora
docker compose --profile neo4j up -d                 # Enable GraphRAG
docker compose --profile langfuse up -d              # Enable self-hosted Langfuse
docker compose --profile qdrant up -d                # Switch to Qdrant
docker compose --profile odl-hybrid up -d --build odl-hybrid  # Docling hybrid (as needed)
```

---

## Full workflow (cloud-agnostic)

```
1) Buy/provision a clean Linux instance on the target cloud (4C8G+ recommended, Ubuntu 22.04)
2) SSH in, copy this directory over, run prepare.sh
3) Verify functionality in the browser
4) Run cleanup.sh (wipes secrets + SSH keys, auto-shuts down)
5) In the cloud console, "build an image / create a snapshot / create an AMI"
6) Create a test instance from the new image, verify firstboot works correctly
7) Share / publish the image (see the platform-specific docs)
```

### Step 1: Deploy on a clean instance

Requirements: systemd + network access + sudo privileges. Ubuntu 22.04 / Debian 12 / CentOS Stream 9 recommended.

**1. Copy the scripts over (pick one method — none require cloning the whole WeKnora repo).**

> The commands need to write to `/opt/`, so the simplest approach is to `sudo -i` into root first and then paste them.
> If you insist on prefixing every line with `sudo`, note that `>>` redirection runs in your current shell — you'll need `sudo tee -a` instead.

> **If you're on a mainland China cloud host, skip straight to Method C (scp)**. In practice, Tencent Cloud / Alibaba Cloud Lighthouse instances often can't reach `github.com`, `raw.githubusercontent.com`, or even overseas / public-good proxies like `gh-proxy.com` (TLS RST or timeouts) — Methods A/B will fail across the board. Since this repo is already on your local machine, scp'ing it over is the most reliable option.

```bash
sudo -i      # switch to root, run subsequent commands directly

# === Method A: sparse checkout (~60KB) ===
# If unreachable, set GH_PROXY=https://gh-proxy.com/ or https://ghfast.top/ (note the trailing slash).
GH_PROXY="${GH_PROXY:-}"
mkdir -p /opt/weknora-tools && cd /opt/weknora-tools
git init -q && git remote add origin "${GH_PROXY}https://github.com/Tencent/WeKnora.git"
git config core.sparseCheckout true
echo "scripts/cloud-image/" >> .git/info/sparse-checkout
git pull -q --depth=1 origin main

# === Method B: direct curl (use this if git isn't available) ===
# If unreachable, set GH_PROXY=https://gh-proxy.com/ or https://ghfast.top/ (note the trailing slash).
GH_PROXY="${GH_PROXY:-}"
mkdir -p /opt/weknora-tools/scripts/cloud-image/systemd && cd /opt/weknora-tools
base="${GH_PROXY}https://raw.githubusercontent.com/Tencent/WeKnora/main/scripts/cloud-image"
for f in prepare.sh cleanup.sh firstboot.sh README.md; do
  curl -fsSL "$base/$f" -o "scripts/cloud-image/$f"
done
for f in weknora.service weknora-firstboot.service; do
  curl -fsSL "$base/systemd/$f" -o "scripts/cloud-image/systemd/$f"
done
chmod +x scripts/cloud-image/*.sh

# === Method C: scp from your local machine (recommended: mainland China cloud hosts should use this directly) ===
# Run this on your local machine (one that can reach GitHub normally):
#   scp -r scripts/cloud-image root@<instance-IP>:/opt/weknora-tools/scripts/
```

> If you're unsure whether the VM can reach a proxy, probe first:
> `for h in gh-proxy.com ghfast.top mirror.ghproxy.com github.moeyy.xyz kkgithub.com; do printf '%-25s' "$h"; curl -sS -o /dev/null -m 5 -w 'http=%{http_code} t=%{time_total}s\n' "https://$h/" 2>&1 || echo FAIL; done`
> Whichever host returns `http=200/301/302`, set `GH_PROXY` to it (with a trailing `/`). If none work, resign yourself to Method C.

**2. Run the deployment:**

```bash
sudo bash /opt/weknora-tools/scripts/cloud-image/prepare.sh

# To pin a specific version (recommended, ensures the image is reproducible)
sudo WEKNORA_REF=v0.5.0 bash /opt/weknora-tools/scripts/cloud-image/prepare.sh

# Mainland China three-piece combo: bypass overseas CDNs for GitHub / get.docker.com / Docker Hub simultaneously
# (Tencent Cloud shown as an example — swap in the corresponding mirrors for Alibaba Cloud / Huawei Cloud)
sudo \
  WEKNORA_REF=v0.5.0 \
  WEKNORA_GH_PROXY=https://gh-proxy.com/ \
  DOCKER_INSTALL_MIRROR=https://mirrors.tencent.com/docker-ce/linux/ubuntu \
  DOCKER_REGISTRY_MIRROR=https://mirror.ccs.tencentyun.com \
  bash /opt/weknora-tools/scripts/cloud-image/prepare.sh
```

> These three variables each address a different overseas-CDN-unreachable problem:
> - `WEKNORA_GH_PROXY`: speeds up **GitHub tarball** downloads (`prepare.sh` step 2, runtime files)
> - `DOCKER_INSTALL_MIRROR`: bypasses **`get.docker.com`**, installing Docker via apt + a docker-ce mirror instead (step 1)
> - `DOCKER_REGISTRY_MIRROR`: speeds up **Docker Hub** image pulls (step 4, `wechatopenai/weknora-*`)
>
> Addresses for different cloud providers (swap the ubuntu/debian portion to match your actual distro):
> | Provider | `DOCKER_INSTALL_MIRROR` | `DOCKER_REGISTRY_MIRROR` |
> |---|---|---|
> | Tencent Cloud | `https://mirrors.tencent.com/docker-ce/linux/ubuntu` | `https://mirror.ccs.tencentyun.com` |
> | Alibaba Cloud | `https://mirrors.aliyun.com/docker-ce/linux/ubuntu` | `https://<your-id>.mirror.aliyuncs.com` |
> | Huawei Cloud | `https://mirrors.huaweicloud.com/docker-ce/linux/ubuntu` | `https://<id>.mirror.swr.myhuaweicloud.com` |
>
> `DOCKER_INSTALL_MIRROR` currently only supports apt-based distros (Ubuntu / Debian / TencentOS-apt).
> CentOS / Rocky and other yum-based distros can generally reach `get.docker.com` directly — revisit if that changes.

`prepare.sh` will:

1. Install Docker / the Docker Compose plugin (skipped if already installed)
2. Use `curl + tar` to download the 4 runtime files to `/opt/WeKnora`
3. Pull and start the default 5 containers + pre-pull the sandbox image
4. Install `weknora.service` (auto-start on boot) + `weknora-firstboot.service` (first-boot init)

Once done, visit `http://<public-IP>`.

### Step 2: Verify

At minimum, verify that you can:

- Register an admin account and log in
- Create a knowledge base
- Upload a document and have it finish parsing
- Perform a Q&A round

```bash
sudo docker compose -f /opt/WeKnora/docker-compose.yml ps
curl -f http://localhost:8080/health
```

### Step 3: Clean up and build the image

> **Important**: `cleanup.sh` deletes all SSH public keys, clears logs, and wipes the database and docker volumes. After running it, **do not SSH in again** — go straight to the cloud console and shut down the instance to build the image.

```bash
sudo bash /opt/weknora-tools/scripts/cloud-image/cleanup.sh
```

The instance will automatically `poweroff` once this finishes. Then follow your cloud provider's docs to build the image from that instance.

### New-instance first-boot behavior

The first time someone boots an instance created from your image, `weknora-firstboot.service` will:

1. Generate random `DB_PASSWORD` / `REDIS_PASSWORD` / `JWT_SECRET` / `SYSTEM_AES_KEY` values
2. Write them back to `/opt/WeKnora/.env`
3. Run `docker compose up -d` to start all services
4. Write the generated credentials to `/root/weknora-credentials.txt` (readable by root only)
5. Disable and delete itself (to guarantee it only runs once)

From then on, `weknora.service` takes over on every subsequent boot.

> **Note**: by default `firstboot.sh` does **not** disable registration (`DISABLE_REGISTRATION=false`) — whoever registers first becomes the admin.
> The credentials file includes a prominent warning to "register promptly to avoid someone else claiming the admin account." For stricter control, add a line `replace DISABLE_REGISTRATION true` to the list of `replace` calls in `firstboot.sh` and rebuild the image.

---

## Upgrading the image version

There's no git repo inside the image — to upgrade, just rerun `prepare.sh` (it overwrites the 4 runtime files, **leaving `.env` and docker volume data untouched**):

```bash
sudo WEKNORA_REF=v0.6.0 bash /opt/weknora-tools/scripts/cloud-image/prepare.sh
sudo bash    /opt/weknora-tools/scripts/cloud-image/cleanup.sh   # before building the new image
```

> **If you want to also clean up old image layers while cutting a new image** (each old weknora-* tag is a few hundred MB — ~2-4GB across 4 images):
> ```bash
> sudo PRUNE_OLD_IMAGES=true WEKNORA_REF=v0.6.0 \
>   bash /opt/weknora-tools/scripts/cloud-image/prepare.sh
> ```
> The default is `false`, to preserve a rollback path. Only enable this once you've confirmed the new version is stable, right before building the image.

## Security notes

- **Do not** bake any LLM API keys, Langfuse keys, or personal SSH keys into the image
- Database / Redis / MinIO ports are only visible to the docker network by default — do not expose them publicly in your cloud firewall
- `/root/weknora-credentials.txt` is created with `umask 077`, readable by root only
- Always run `cleanup.sh` before rebuilding the image, to avoid leaking prior test data / SSH keys / machine-id

## Platform-specific instructions

| Platform | Docs |
|---|---|
| Tencent Cloud Lighthouse / CVM | [`docs/cloud-image/tencent-lighthouse.md`](../../docs/cloud-image/tencent-lighthouse.md) |
| AWS EC2 AMI | (contributions welcome) |
| Alibaba Cloud ECS | (contributions welcome) |
