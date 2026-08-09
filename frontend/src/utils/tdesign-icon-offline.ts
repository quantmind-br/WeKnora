/**
 * Block iconfont requests from tdesign-icons-vue-next to the external CDN.
 *
 * Background (issue #867 / #897):
 * The Icon / IconFont components in tdesign-icons-vue-next, on onMounted, use
 * `checkScriptAndLoad` / `checkLinkAndLoad` to inject into the document a
 * <script> / <link> pointing to
 * `https://tdesign.gtimg.com/icon/<version>/fonts/index.(js|css)`. In an environment without external network access, the request fails, causing all icons to fail to render.
 *
 * Approach: before the Vue app mounts, pre-insert placeholder nodes matching tdesign's matching rules,
 * so that the dedup check in `checkScriptAndLoad` / `checkLinkAndLoad` hits and returns early,
 * preventing it from appending the real node pointing to the CDN.
 *
 * - Matching selector source: tdesign-icons-vue-next/esm/utils/check-url-and-load.js
 *       `.t-svg-js-stylesheet--unique-class[src="<url>"]`
 *       `.t-iconfont-stylesheet--unique-class[href="<url>"]`
 * - The placeholder <script> / <link> uses a non-standard type / rel, so the browser won't actually issue a network request.
 *
 * The SVG sprite symbols are registered locally in advance via the <script src="/tdesign-icons/.../index.js"></script>
 * tag in index.html, so <t-icon name="..."> still renders correctly.
 */

const SVG_SCRIPT_CLASS = "t-svg-js-stylesheet--unique-class";
const ICONFONT_LINK_CLASS = "t-iconfont-stylesheet--unique-class";

// Aligns with the addresses hardcoded internally in tdesign-icons-vue-next 0.4.x; multiple version numbers are kept in parallel to accommodate potential upgrades.
const BLOCKED_ICON_VERSIONS = ["0.4.0", "0.4.1", "0.4.2", "0.4.3", "0.4.4"];

const BLOCKED_SCRIPT_URLS = BLOCKED_ICON_VERSIONS.map(
  (version) => `https://tdesign.gtimg.com/icon/${version}/fonts/index.js`,
);

const BLOCKED_LINK_URLS = BLOCKED_ICON_VERSIONS.map(
  (version) => `https://tdesign.gtimg.com/icon/${version}/fonts/index.css`,
);

let installed = false;

export function installTDesignIconOfflineGuard(): void {
  if (installed || typeof document === "undefined") return;
  installed = true;

  const body = document.body;
  if (!body) {
    document.addEventListener(
      "DOMContentLoaded",
      () => installTDesignIconOfflineGuard(),
      { once: true },
    );
    installed = false;
    return;
  }

  BLOCKED_SCRIPT_URLS.forEach((src) => {
    const exists = document.querySelector(
      `script.${SVG_SCRIPT_CLASS}[src="${src}"]`,
    );
    if (exists) return;
    const stub = document.createElement("script");
    stub.setAttribute("class", SVG_SCRIPT_CLASS);
    stub.setAttribute("src", src);
    // A non-standard MIME type makes the browser skip the fetch/execution phase of the script
    stub.setAttribute("type", "text/no-load");
    stub.setAttribute("data-weknora-blocked-cdn", "tdesign-icons");
    body.appendChild(stub);
  });

  BLOCKED_LINK_URLS.forEach((href) => {
    const exists = document.querySelector(
      `link.${ICONFONT_LINK_CLASS}[href="${href}"]`,
    );
    if (exists) return;
    const stub = document.createElement("link");
    stub.setAttribute("class", ICONFONT_LINK_CLASS);
    stub.setAttribute("href", href);
    // Without rel="stylesheet" declared, the browser won't issue a stylesheet request
    stub.setAttribute("rel", "preload-blocked");
    stub.setAttribute("data-weknora-blocked-cdn", "tdesign-icons");
    document.head.appendChild(stub);
  });
}
