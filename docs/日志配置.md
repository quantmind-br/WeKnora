# Log Configuration

WeKnora's logging is managed centrally by `internal/logger`, with all behavior driven by environment variables — no code changes required.

## Environment Variables

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| `LOG_LEVEL` | No | `debug` | Log level, one of `debug` / `info` / `warn`(`warning`) / `error` / `fatal`; invalid values fall back to `debug` |
| `LOG_PATH`  | No | Empty (stdout only; in macOS `.app` bundle mode, falls back to `~/Library/Logs/<AppName>/<AppName>.log`) | On-disk path — when enabled, writes to both stdout and this file, with the file rotated via lumberjack (50MB per file / 3 backups / 28 days / compressed archives) |
| `LOG_FORMAT` | No | Empty (uses the built-in default format) | Custom log output template, supports the placeholders below |

> ANSI colors are enabled automatically in terminal environments; in non-TTY contexts (container log collection, `docker logs` redirection, etc.) colors are automatically disabled to avoid issues with log aggregation/search.

## `LOG_FORMAT` Template

When `LOG_FORMAT` is empty, the built-in default format is used (consistent with prior behavior):

```
INFO [2026-05-21 10:20:30.123] [req-abc k1=v1] file.go:42[fn] | message body
```

Setting `LOG_FORMAT` enables template mode, which supports the following placeholders:

| Placeholder | Meaning |
| --- | --- |
| `%d`       | Timestamp (`2006-01-02 15:04:05.000`) |
| `%level`   | Log level (`DEBUG` / `INFO` / `WARNING` / `ERROR` / `FATAL`); when color is enabled, only this placeholder is colorized |
| `%thread`  | Current goroutine ID. **`runtime.Stack` is not invoked unless this placeholder is referenced, so there is no extra overhead when it's unused** |
| `%logger`  | Caller info (`file.go:line[func]`); truncated to the last 50 characters if too long |
| `%traceId` | Request ID (i.e., `request_id` from the context) |
| `%msg`     | Message body + remaining structured fields (`key=value`, concatenated in ascending key order) |

Example:

```bash
export LOG_FORMAT='[%d] %level %thread %logger %traceId | %msg'
```

Output (with terminal colors enabled, only the `%level` segment is colorized):

```
[2026-05-21 10:20:30.123] INFO 17 service.go:88[Handle] req-abc | hello extra=ok
```

### Implementation Details and Notes

- Placeholder substitution uses a single-pass scan via `strings.NewReplacer` — even if a preceding placeholder's value contains another placeholder's literal string (e.g. `request_id=%msg`), it will not be substituted a second time.
- Level colorization injects ANSI colors directly during the `%level` substitution step, and will **not** mistakenly colorize literal occurrences of strings like `INFO` / `ERROR` in the message body.
- Structured fields (`logger.WithField` / `WithFields`) are appended after `%msg`; template mode does not currently support extracting arbitrary fields into their own placeholders — if you need K/V structured search, disable `LOG_FORMAT` and use the default format instead.
- Changes to `LOG_LEVEL` and `LOG_PATH` take effect via `logger.ConfigureFromEnv()`, re-applied after `main` loads `.env`.
