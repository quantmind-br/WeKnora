# Documentation has moved to website-docs

Product, deployment, API and development documentation is maintained in [website-docs](../website-docs/README.md). Legacy hand-written documents that were migrated, duplicated or outdated have been deleted; their history is available in Git.

- [FAQ and upgrade troubleshooting](../website-docs/01-getting-started/05-troubleshooting.md)
- [API reference](../website-docs/04-api/01-api-overview.md)
- [Development guide](../website-docs/06-development/01-dev-guide.md)
- [Migration mapping and remaining dependencies](../website-docs/MIGRATION.md)

This directory only keeps the following engineering resources and this entry note:

- `docs.go`, `swagger.json`, `swagger.yaml` and the contract tests: part of the backend build and tests; the generated files are still updated by `make docs`.
- `LITE.md`: copied into the Lite release package as the offline README.
- `images/`, `assets/`: image resources still used by the README, Helm, etc.
- `poc/docker-sandbox/`: a standalone Go experiment module; its source and run instructions are kept, but it is not a current product guide.

These resources are still used for builds, releases or historical experiments, so the directory cannot simply be deleted. Do not put new product documentation here.
