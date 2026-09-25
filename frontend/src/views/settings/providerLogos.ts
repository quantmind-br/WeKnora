// Lookup table for the logo on the left side of the settings card.
//
// Assets are split into two categories:
// color/ —— official multi-color vendor SVGs, rendered directly via <img>, keeping brand colors
// mono/  —— single-color SVGs (mostly from simple-icons / vendor marks), tinted via mask-image with
// the card's own brand color, thereby following the low-saturation brand tones
// defined by rules like .store-card--<id>.
//
// Callers pass (category, id) to get { mode, url }; returns undefined when not found,
// the card falls back to the original first-letter monogram.

const colorModules = import.meta.glob('@/assets/img/providers/color/*/*.svg', {
  eager: true,
  query: '?url',
  import: 'default',
}) as Record<string, string>;

const monoModules = import.meta.glob('@/assets/img/providers/mono/*/*.svg', {
  eager: true,
  query: '?url',
  import: 'default',
}) as Record<string, string>;

export type ProviderCategory = 'vectorstore' | 'storage' | 'websearch' | 'parser' | 'sandbox';

export type LogoMatch = {
  mode: 'color' | 'mono';
  url: string;
};

const buildLookup = (modules: Record<string, string>, segment: string) => {
  const map: Partial<Record<ProviderCategory, Record<string, string>>> = {};
  const re = new RegExp(`providers/${segment}/([^/]+)/([^/]+)\\.svg$`);
  for (const [path, url] of Object.entries(modules)) {
    const match = path.match(re);
    if (!match) continue;
    const [, category, id] = match;
    const bucket = (map[category as ProviderCategory] ||= {});
    bucket[id.toLowerCase()] = url;
  }
  return map;
};

const colorLookup = buildLookup(colorModules, 'color');
const monoLookup = buildLookup(monoModules, 'mono');

export function providerLogo(
  category: ProviderCategory,
  id: string | undefined | null,
): LogoMatch | undefined {
  if (!id) return undefined;
  const key = id.toLowerCase();
  const colorUrl = colorLookup[category]?.[key];
  if (colorUrl) return { mode: 'color', url: colorUrl };
  const monoUrl = monoLookup[category]?.[key];
  if (monoUrl) return { mode: 'mono', url: monoUrl };
  return undefined;
}
