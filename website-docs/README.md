# WeKnora Website and Documentation

`website-docs/` contains the website, the documentation, shared styles, and all build and deployment scripts. It can be copied and built on its own, without depending on files outside this directory. The website and the documentation share one domain, one build, and one deployment artifact.

Product, deployment, API, and development documentation are all maintained here. The still-relevant content of the old `docs/` has been merged by topic, and outdated or duplicated text has been removed; see the [migration record](MIGRATION.md) for the migration decisions and the engineering resources that were kept. That record is for maintainers only and is not published to the site.

- `/`: product homepage.
- `/docs/`: goes straight to "Quick Start".
- `/docs/…`: full documentation, with category navigation, search, and sidebar.

The site only occupies these two paths: the root contains only `index.html`, `404.html`, and `docs/`. The website's scripts, styles, and images all live under `/docs/_home/`, so an upstream gateway only needs to forward `/` (exact match) and the `/docs/` prefix. `build` rejects any page that references resources outside these two paths.

The top bar is 64px tall and spans the full width. Both sides use the original README logo and share colors, fonts, and the light/dark preference; the logo returns to the website in the current tab. The website and the documentation each use navigation suited to their own pages.

## Publishing: Static Files + Nginx

### 1. Prepare the Deployment Package

Use Node.js 24 and run the following in the repository's `website-docs/` directory. On the first build, or after a dependency lock file changes, install dependencies first:

```bash
cd website-docs
npm run setup
npm run build
npm run package:site
```

`build` compiles the website, checks documentation links and Mermaid diagrams, compiles the documentation, and validates the merged resource paths. When everything passes, it updates `static-site/`; `package:site` packages it into `releases/`.

### 2. Upload and Extract

Only the archive needs to be uploaded to the server. The server needs neither Node.js nor the WeKnora backend or a separate documentation service.

Use a new, empty directory for every release to avoid mixing in files from an old site. `20260915-1` below is an example release number; use a different number for each later release:

```bash
sudo mkdir -p /srv/www/weknora/releases/20260915-1
sudo tar -xzf weknora-site-v0.8.2.tar.gz -C /srv/www/weknora/releases/20260915-1
```

After extraction, the directory should directly contain `index.html`, `404.html`, and `docs/`, with no extra `static-site/` level.

### 3. Configure the Domain Root

Add a site configuration using [deploy/nginx.conf](deploy/nginx.conf) as a template, and change:

```nginx
server_name your-domain;
root /srv/www/weknora/releases/20260915-1;
```

If the domain already has an HTTPS configuration, keep the existing certificate and listen settings, and merge the template's `root`, `index`, and `location` rules into the existing site configuration. The domain should point to this server.

The `.html` route resolution rules for the documentation must be kept; do not fall back every unknown path to the website's `index.html`. The current build is deployed at the domain root and does not support being placed directly in a subdirectory such as `/weknora/`.

Check the configuration, then reload:

```bash
sudo nginx -t
sudo nginx -s reload
```

### 4. Verify the Release

Visit the following paths:

- `/`: the website.
- `/docs/`: Quick Start.
- `/docs/03-features/23-memory`: the long-term memory documentation; opening it directly and refreshing both work.
- `/docs/not-found`: returns 404.

Then confirm that search, the light/dark toggle, and the logo link back to the website all work.

For later updates, repeat build, package, and upload to a new directory, then change the Nginx `root` and reload. To roll back, point `root` back to the previous release directory. Clean up old release directories only after the new version is confirmed stable.

## Optional: Docker Deployment

It builds from clean source, with no need to install Node.js on the host or pre-generate static files. The Dockerfile uses two stages: Node.js 24 installs dependencies from the two lock files and builds the website and documentation, and the final Nginx image contains only the static output and the server configuration.

Run from the repository root:

```bash
docker build -t weknora-site:0.8.2 website-docs
docker run -d --name weknora-site --restart unless-stopped -p 8080:80 weknora-site:0.8.2
```

Visit `http://server-address:8080/`. If you use a domain and HTTPS, have your existing reverse proxy forward to this port. The image already contains the website, the documentation, and the Nginx routing configuration.

Nginx inside the container listens on port 80 by default; you can change it with the `WEBSITE_NGINX_PORT` environment variable without rebuilding. Set it like this when using `--network host`, or when your deployment platform requires the container to listen on a specific port:

```bash
docker run -d --name weknora-site --restart unless-stopped -e WEBSITE_NGINX_PORT=8080 -p 8080:8080 weknora-site:0.8.2
```

If you only want a different external port, changing the host port on the left side of `-p` is enough, for example `-p 9000:80`.

You can also copy just the `website-docs/` directory and run `docker build -t weknora-site:0.8.2 .` in it. The build context must be `website-docs/`; host dependencies, old build output, and deployment packages are excluded by `.dockerignore`.

When migrating from the old documentation image, change the container port mapping or the reverse proxy target port from `8081` to `80` (the host port is up to you, for example `-p 8081:80`). The domain root `/` now serves the website and `/docs/` serves the documentation, so the reverse proxy must cover the whole site and preserve the request path. The container uses the default entrypoint of the official Nginx image; no extra `docker-entrypoint.sh` is needed.

After building, you can run the following check on a machine with Node.js 24 and Docker installed, without installing npm dependencies. The check starts a temporary container, cleans it up automatically, and verifies routing for both sites, static resources, 404, compression, and response headers:

```bash
cd website-docs
npm run test:docker -- weknora-site:0.8.2
```

## Local Development

Use Node.js 24 LTS (the directory provides `.nvmrc`; with nvm you can run `nvm use`). After getting the source for the first time, install dependencies (installed separately from the documentation and website lock files):

```bash
cd website-docs
npm run setup
npm run build
npm run preview
```

Preview address: `http://127.0.0.1:3000/`. After changing the source, rebuild and refresh; use `npm run preview -- --port 8080` to pick another port.

```bash
npm run check
npm run test:integration
```

The integration check requires Google Chrome and verifies theme sync, navigation on both sides, search, mobile controls, and direct documentation routes. Use `SITE_TEST_URL` to specify the site to verify.

During development, use `npm run dev:homepage` (website) and `npm run dev:docs` (documentation) separately for hot reload; for complete in-site navigation, use `npm run preview` after a unified build.

`npm run build:docs` compiles only the documentation and `npm run build:homepage` compiles only the website; official releases use `npm run build`, whose output includes both. `VERSION` is the release version used by this site build.

## Project Layout (relative to website-docs)

- `homepage/`: Next.js website source, brand assets, and its own dependency lock file.
- `01-getting-started/` to `06-development/`: documentation content; the original paths are unchanged.
- `.vitepress/`: documentation site configuration and theme.
- `public/`, `sample-data/`: documentation screenshots and samples.
- `shared/`: brand variables, top bar styles, and shared icons.
- `scripts/`: unified build, checks, preview, and packaging.
- `static-site/`: the single deployment directory, generated by the build.
- `deploy/`, `Dockerfile`: Nginx and Docker deployment configuration.
- `releases/`: generated deployment archives.

For the sources of the brand assets, see [homepage/BRAND-ASSETS.md](homepage/BRAND-ASSETS.md). The product video uses the original GitHub attachment from the README, so playback requires access to GitHub; pages, documentation, and screenshots are deployed with the package.
