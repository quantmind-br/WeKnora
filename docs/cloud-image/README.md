# WeKnora Cloud Image Packaging Guide

Package WeKnora into a distributable cloud image (AMI / custom image / snapshot) so users can spin up an instance from the image and have it ready to use on boot, with automatically randomized secrets and zero private data leakage.

## Common Tooling

Cloud-agnostic scripts and detailed instructions: [`scripts/cloud-image/README.md`](../../scripts/cloud-image/README.md)

Includes the `prepare.sh` / `cleanup.sh` / `firstboot.sh` scripts and two systemd units, verified across multiple distributions (Ubuntu / Debian / CentOS / TencentOS).

## Platform-Specific Instructions

| Platform | Documentation | Status |
|---|---|---|
| Tencent Cloud Lighthouse / CVM | [tencent-lighthouse.md](./tencent-lighthouse.md) | ✅ |
| AWS EC2 (AMI) | _Contributions welcome_ | ⏳ |
| Alibaba Cloud ECS | _Contributions welcome_ | ⏳ |
| Volcano Engine ECS | _Contributions welcome_ | ⏳ |
| Huawei Cloud ECS | _Contributions welcome_ | ⏳ |
| Local KVM / Proxmox | _Contributions welcome_ | ⏳ |

> Try to keep the document structure consistent across platforms: recommended instance specs → image-creation steps → sharing/publishing method → platform-specific caveats.
