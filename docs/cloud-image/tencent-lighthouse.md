# Tencent Cloud Lighthouse / CVM Image Building Guide

> **Prerequisite reading**: for the general scripts and workflow, see [`scripts/cloud-image/README.md`](../../scripts/cloud-image/README.md). This document only covers the Tencent Cloud–specific steps.

## Recommended Instance Specs

| Item | Recommended Value |
|---|---|
| Instance Type | Lighthouse Instance / Standard CVM |
| CPU / Memory | At least 4 cores / 8 GB (4 cores / 16 GB recommended for running full RAG + Agent features) |
| System Disk | At least 80 GB SSD |
| Image | Ubuntu Server 22.04 LTS / TencentOS Server 3.1 |
| Region | Choose the region where your target users are mainly located (cross-account sharing only works within the same region) |

## Full Workflow

1. Purchase an instance matching the specs above from the console
2. SSH into it and run `prepare.sh` following [scripts/cloud-image/README.md](../../scripts/cloud-image/README.md)
3. Open `http://<public-IP>` in a browser to verify functionality
4. Run `cleanup.sh` (this will automatically `poweroff`)
5. Go to the console and **"Create Image"** (see below)
6. Create a test instance from the new image and verify that firstboot works correctly
7. **"Share Image"** with other accounts / apply to list it on the **"Image Marketplace"** (see below)

## Creating an Image

**Lighthouse Instance**:

1. Console → Lighthouse Instances → select the powered-off instance
2. "More" → **"Create Image"**
3. It's recommended to include a version number in the image name: `weknora-v0.5.0-ubuntu2204`
4. Wait 5–30 minutes (depending on the system disk size)

**CVM**:

1. Console → Cloud Virtual Machines → select the powered-off instance
2. "More" → "Create Image" → select "Full Image"
3. It's likewise recommended to include a version number

> Under the same account, custom images are subject to a quantity quota (default 20), which can be viewed in the console.

## Verifying the Image

It is strongly recommended to create a test instance from the new image and verify at least the following:

- [ ] You can SSH into it (using the console's default password / your imported key)
- [ ] `systemctl status weknora-firstboot` shows it executed successfully (or has been disabled + the file deleted)
- [ ] `cat /root/weknora-credentials.txt` contains a randomly generated password
- [ ] Opening the public IP in a browser lets you access WeKnora and register an administrator
- [ ] `docker compose -f /opt/WeKnora/docker-compose.yml ps` shows everything as healthy
- [ ] `cat /opt/WeKnora/.cloud-image-meta` shows the correct version

## Sharing With Other Users

There are 3 methods, listed in increasing order of "scope of reach":

### Method A: Cross-Account Sharing (Private Sharing)

Console → Custom Images → select the image → **"Share"** → enter the recipient's Tencent Cloud account ID (UIN).

- The recipient will see it in their own "Shared Images" list and can create instances directly from it
- **Limitations**: must be in the same region; the recipient's account must already have the corresponding product (Lighthouse / CVM) enabled
- Suitable for small-scale sharing, partners, or beta testers

### Method B: Cross-Region Use

A Lighthouse image can be "shared to CVM," and once converted into a CVM custom image, it can then:

- Be copied across regions
- Be exported as a `qcow2` file and downloaded locally (more general-purpose, usable with KVM / other clouds)

### Method C: Listing an Image Product on the Tencent Cloud Marketplace (Public One-Click Deployment)

This is the format that truly lets "any user find it on the Cloud Marketplace and one-click purchase/deploy it." The process is fairly involved and is suited to long-term operation.

Reference documentation (defer to the official docs):

- [Cloud Marketplace – Image Service Listing Process](https://cloud.tencent.com/document/product/306/3019)
- [Cloud Marketplace – Image Product Creation Guide](https://cloud.tencent.com/document/product/306/30128)
- [Lighthouse – Application Image Usage Guide](https://cloud.tencent.com/document/product/1207/72665)
- [Lighthouse – Managing Images (How-To Guide)](https://cloud.tencent.com/document/product/1207/63263)

Rough steps:

1. Log in to the [Tencent Cloud Marketplace Service Provider Console](https://console.cloud.tencent.com/serviceprovider) and register as a service provider
2. Go to "Product Management → Product List → New Product," and select "**Image**" as the access type
3. Select the custom image you've already created (i.e., the one produced via Method A/B above)
4. **Complete a Host Security (Pro Edition) scan** — this is a mandatory prerequisite for listing an image, requiring you to pay for CVM + Host Security Pro Edition usage yourself; it's recommended to use pay-as-you-go billing and release the resources once the scan is done
5. Fill in the product name, version, highlights, details, usage guide (a PDF/Word/PPT/ZIP/RAR file is required, ≤2MB), and after-sales support information
6. Choose a pricing model (pay-as-you-go / subscription); pay-as-you-go currently only supports the free (0-cost) tier
7. Submit for review (Cloud Marketplace operations staff review takes about 7 business days)
8. Once approved, users will be able to select your image on the Cloud Marketplace or when purchasing a CVM

> WeKnora is a Tencent-affiliated open-source project (`Tencent/WeKnora`). If you'd like to push for an official listing, it's recommended to contact the maintainer team via [WeKnora GitHub Issues](https://github.com/Tencent/WeKnora/issues) rather than applying individually on your own.

## Notes

- Tencent Cloud has good cloud-init compatibility, so `cloud-init clean` in `cleanup.sh` works correctly
- Lighthouse images default to a size limit equal to the system disk size, so plan ahead accordingly
- If using a domain name + HTTPS, it's recommended to configure certificates separately via [acme.sh](https://acme.sh) / certbot outside of firstboot, rather than baking the certificate into the image
