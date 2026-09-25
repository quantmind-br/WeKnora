# Docker sandbox backend feasibility PoC

For the current implementation and deployment requirements, see [Sandbox deployment and troubleshooting](../../../website-docs/06-development/04-sandbox-deployment.md). This PoC only preserves a historical feasibility experiment and does not reflect the current adapter behavior.

This program talks to the Docker Engine API directly and checks, item by item, "whether Docker can carry the WeKnora
`RemoteSandboxClient` contract + the E2B-style snapshot workflow". It is not product code and is not part of the main module build
(it has its own `go.mod`); it only serves as reproducible evidence for the research findings.

Every item prints `PASS/FAIL` and the actual observed value. Steps marked `(GAP)` assert "what Docker cannot do",
and they also end in PASS: PASS means the gap was reproduced, not that the capability exists.

## How to run

You need a reachable Docker daemon (`DOCKER_HOST` is honored) that can pull `python:3.11-slim`:

```bash
cd docs/poc/docker-sandbox
go run .            # may need sudo -E when the daemon is on a local unix socket
```

The program builds its own template image with a uid 1000 `user` account (matching the E2B template convention) and deletes the containers it created when done.
Clean up the leftover `weknora-poc/*` images with `docker image rm`.

## What it covers

- Lifecycle: Create / Connect (reconnect with a new client) / List (label filter) / Delete / Pause / Stop+Start
- Execution: user, workdir, env, stdin, separate stdout+stderr, exit code, timeout
- File surface: reads, writes and stat through the archive API; mkdir/ls/rm implemented with exec
- Session semantics: `pip install` and written files survive across execs
- Snapshot: commit to an image, start a new sandbox from a snapshot, v1→v2 increments, list by label, layer count and size
- Gap reproduction: client cancellation does not kill the process, root cannot bypass permission bits after CapDrop ALL, snapshots do not include memory state,
  the daemon has no idle TTL, killing `docker run` leaves the container running
