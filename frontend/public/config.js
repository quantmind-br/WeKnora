// Runtime config (local dev defaults; overridden by the entrypoint script in Docker)
window.__RUNTIME_CONFIG__ = {
  MAX_FILE_SIZE_MB: 50,
  // Optional: serve embed on a dedicated origin, e.g. 'https://embed.example.com'
  EMBED_BASE_URL: '',
};
