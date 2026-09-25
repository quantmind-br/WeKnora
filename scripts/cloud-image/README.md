# Cloud Image Maintenance Scripts

Script scope, image-building caveats, and first-boot troubleshooting are maintained in the [website-docs development guide](../../website-docs/06-development/01-dev-guide.md#cloud-image-scripts). For a regular installation, see [Installation](../../website-docs/01-getting-started/02-installation.md).

This directory keeps `prepare.sh`, `cleanup.sh`, `firstboot.sh`, and their systemd units. `cleanup.sh` wipes the build machine's data, secrets, SSH authorizations, and caches, then shuts it down, so only use it on a dedicated build machine; the scripts themselves are the source of truth for parameters and actual behavior.
