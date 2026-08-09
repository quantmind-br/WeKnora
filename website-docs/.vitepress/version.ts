import { existsSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const versionFile = resolve(import.meta.dirname, '../../VERSION')

/** Release version from the VERSION file at the repo root (same source as scripts/get_version.sh) */
export function getRepoVersion(): string {
  if (!existsSync(versionFile)) return 'unknown'
  return readFileSync(versionFile, 'utf-8').trim() || 'unknown'
}

export const repoVersion = getRepoVersion()

export const repoVersionLabel =
  repoVersion === 'unknown' || repoVersion.startsWith('v') ? repoVersion : `v${repoVersion}`
