# Smart Home Hub Pro Product Manual

**Product Model**: HUB-Pro-2024
**Document Version**: v2.1
**Applicable Firmware**: ≥ 3.4.0

## 1. Product Overview

The Smart Home Hub Pro is a home IoT hub launched by NovaTech, responsible for the unified management of lighting, air conditioning, curtains, security sensors, and other devices. Users can complete scene automation via the mobile app, voice assistant, or local touchscreen.

Core selling points:

- Local offline availability: configured scenes can still run over the local network after losing internet access
- Multi-protocol compatibility: simultaneous support for Matter, Zigbee 3.0, Wi-Fi, and Bluetooth Mesh
- Edge AI: built-in lightweight model that can recognize colloquial commands such as "I'm home" and "Ready for a movie"

## 2. Technical Specifications

| Item | Parameter |
| --- | --- |
| Processor | Quad-core ARM Cortex-A55, 1.8 GHz |
| Memory / Storage | 4 GB RAM / 32 GB eMMC |
| Wireless Protocols | Wi-Fi 6, Zigbee 3.0, Bluetooth 5.2, Thread |
| Wired Interfaces | 1× Gigabit Ethernet, 1× USB-C (debugging) |
| Power Supply | DC 12V / 2A, typical power consumption 8W |
| Operating Temperature | 0℃ ~ 40℃ |
| Maximum Connected Devices | 256 (recommended ≤ 120 to ensure response speed) |

## 3. Initial Setup

1. Connect the device to power; the indicator light will slowly blink blue, indicating it is waiting for network configuration.
2. Open the NovaHome App, select "Add Hub," and scan the QR code on the device body.
3. Follow the wizard to connect to your home Wi-Fi; if using a wired network, you may skip the Wi-Fi step.
4. Complete the firmware check; if an update is available, it is recommended to upgrade before adding sub-devices.

> **Note**: During initial network configuration, the phone must be on the same 2.4 GHz band as the hub. 5 GHz-only routers must first enable 2.4 GHz compatibility mode.

## 4. Scene Automation Examples

### 4.1 "Home Mode"

Trigger conditions (any one is sufficient):

- Phone GPS enters the home geofence
- Front door fingerprint lock is unlocked
- Voice command "I'm home"

Actions performed:

- Living room main light adjusted to 70% warm white
- Air conditioner set to 26℃ cooling (only active in the summer template)
- Security arming disabled

### 4.2 "Away Mode"

Trigger condition: All family members' phones leave the geofence for more than 5 minutes.

Actions performed: Turn off all lights in the house, turn off the air conditioner, activate security arming, and shut off the gas valve actuator (if connected).

## 5. FAQ

**Q: Can voice control still be used after the hub goes offline?**
A: If the voice module relies on cloud-based recognition, only the App and local touchscreen are supported once external internet access is lost; if a local voice pack is configured, basic commands can still be used.

**Q: How many hubs can be bound to one account?**
A: Up to 3 for the personal edition; the enterprise edition is licensed per contract, defaulting to 50.

**Q: What is the warranty policy?**
A: The whole unit is covered for 24 months, and battery-type accessories for 12 months. Manual disassembly and water damage are not covered under warranty.

## 6. Related Documents

- For installation and wiring instructions, see the "Hub Pro Hardware Installation Guide"
- For reimbursement and business travel equipment rules, see the "Employee Handbook · Reimbursement and Assets"
- For the Q1 feature roadmap, see the "Product Meeting Minutes · Q1 Planning"
