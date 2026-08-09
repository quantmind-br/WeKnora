Vou traduzir o documento diretamente, preservando toda a estrutura markdown.

---
title: Agent Skills System
tags: [Core Features, Agent, Skills, Sandbox]
aliases: [Agent Skills, Skills System, agent-skills]
source: agent-skills.md
---

# Agent Skills System

## Overview

Agent Skills is an extension mechanism that lets an Agent learn new capabilities by reading an "instruction manual." Unlike traditional hardcoded tools, Skills extend the Agent's capabilities by injecting content into the System Prompt, following the **Progressive Disclosure** design philosophy. Currently only supported by agents with **intelligent reasoning** capability.

### Core Features

- **Non-intrusive extension**: Does not affect the original Agent ReAct flow
- **On-demand loading**: Three-tier progressive loading, optimizing token usage
- **Sandboxed execution**: Scripts run safely in an isolated environment
- **Flexible configuration**: Supports multiple directories and whitelist filtering

> Skills and [MCP](../Core-Features/MCP-Usage-Guide.md) are two different Agent extension mechanisms: Skills work through prompt injection, while MCP works through protocol calls to external tools.

## Design Philosophy

### Progressive Disclosure

```
┌─────────────────────────────────────────────────────────────────┐
│ Level 1: Metadata                                      │
│ • Always loaded into the System Prompt • ~100 tokens/skill                  │
│ • Contains: skill name + short description                                       │
└─────────────────────────────────────────────────────────────────┘
                              ↓ When a user request matches
┌─────────────────────────────────────────────────────────────────┐
│ Level 2: Instructions                                    │
│ • Loaded on demand via the read_skill tool • SKILL.md instruction content              │
│ • Contains: detailed instructions, code examples, usage methods                               │
└─────────────────────────────────────────────────────────────────┘
                              ↓ When more information is needed
┌─────────────────────────────────────────────────────────────────┐
│ Level 3: Additional Resources                                   │
│ • Load specific files via the read_skill tool                               │
│ • Execute scripts via execute_skill_script                            │
└─────────────────────────────────────────────────────────────────┘
```

## Skill Directory Structure

```
my-skill/
├── SKILL.md           # Required: main file (with YAML frontmatter)
├── REFERENCE.md       # Optional: supplementary documentation
├── templates/         # Optional: template files
└── scripts/           # Optional: executable scripts
```

## Preloaded Skills

The system comes with the following 5 built-in preloaded skills:

| Skill | Purpose |
|------|------|
| citation-generator | Automatically generates standardized citation formats |
| data-processor | Data processing and analysis |
| doc-coauthoring | Guides users through structured document authoring |
| document-analyzer | Deep analysis of document structure and content |
| summary-generator | Content summary generation |

Preloaded skills are located in the `skills/preloaded/` directory.

## Sandbox Security Mechanism

### Script Security Validation

Multiple layers of security validation are performed before execution: dangerous command detection, dangerous pattern matching, network access detection, reverse shell detection, argument injection detection, and more.

### Sandbox Modes

| Mode | Description |
|------|------|
| `docker` | Uses Docker container isolation (recommended) |
| `local` | Local process execution (basic security restrictions) |
| `disabled` | Disables script execution |

Configured via the `WEKNORA_SANDBOX_MODE` environment variable.

## Configuration Example

```json
{
  "skills_enabled": true,
  "skill_dirs": ["/path/to/project/skills"],
  "allowed_skills": ["pdf-processing", "code-review"]
}
```

## Related Topics

- [MCP Feature Usage Guide](MCP-Usage-Guide.md) — Another Agent extension mechanism
- [IM Integration Development](../Integration-Extension/IM-Integration-Development.md) — Agents can use skills through IM channels
- [Development Guide](../Development-Deployment/Development-Guide.md) — Building the sandbox image

---

## Backlinks

- [Home](../Home.md) — Wiki home navigation
- [MCP Feature Usage Guide](MCP-Usage-Guide.md) — Agent extension mechanism alongside Skills
- [IM Integration Development](../Integration-Extension/IM-Integration-Development.md) — Agents can use skills in IM channels
- [Version Roadmap](../Project-Overview/Version-Roadmap.md) — Skills community extension direction in the roadmap

---

Tradução completa entregue acima, com toda a estrutura markdown preservada (frontmatter, tabelas, blocos de código, links).
