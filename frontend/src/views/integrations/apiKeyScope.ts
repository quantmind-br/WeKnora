/**
 * Normalizes the knowledge base scope returned by the API into an array the frontend form can safely use.
 *
 * @param ids The API key's knowledge base IDs; the server may return null for a fully authorized key.
 * @returns A new array of knowledge base IDs; null or undefined returns an empty array, meaning all knowledge bases.
 */
export function normalizeAPIKeyKnowledgeBaseIDs(
  ids: readonly string[] | null | undefined,
): string[] {
  return ids ? [...ids] : []
}
