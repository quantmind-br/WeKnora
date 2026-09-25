# Skills Examples

This directory contains examples of the Agent Skills feature.

## Directory Structure

```
skills/
├── README.md              # This file
└── pdf-processing/        # PDF processing skill example
    ├── SKILL.md           # Main file (Level 2)
    ├── FORMS.md           # Supplementary documentation (Level 3)
    └── scripts/           # Executable scripts
        ├── analyze_form.py
        └── extract_text.py
```

## Quick Start

### Running the Demo

```bash
go run ./cmd/skills-demo/main.go
```

### Creating a New Skill

1. Create a new folder in this directory:

```bash
mkdir my-new-skill
```

2. Create `SKILL.md`:

```markdown
---
name: my-new-skill
description: Description of what this skill does and when to use it.
---

# My New Skill

Instructions for the agent...
```

3. Add scripts (optional):

```bash
mkdir my-new-skill/scripts
# Add your scripts
```

## Detailed Documentation

For complete documentation, see: [Agent Skills Documentation](../../website-docs/03-features/22-skills-sandbox.md)

## Example: pdf-processing

This is a fully functional example skill that demonstrates:

- **SKILL.md**: Main file containing YAML frontmatter
- **FORMS.md**: Supplementary reference documentation
- **scripts/**: Python scripts that can be executed in a sandbox

### Skill Description

```yaml
name: pdf-processing
description: Extract text and tables from PDF files, fill forms, merge documents.
```

### Included Scripts

| Script | Function |
|------|------|
| `analyze_form.py` | Analyze PDF form fields |
| `extract_text.py` | Extract text from PDF |

### Usage Example

The Agent automatically invokes based on the user's request:

```
User: "Analyze this PDF form and tell me what fields it has"

Agent: 
  1. Identifies a match for the pdf-processing skill
  2. Calls read_file(path="skill://pdf-processing/SKILL.md") to load the skill content
  3. Calls shell_exec(skill_name="pdf-processing", command=...) to run analyze_form.py
  4. Returns the form field analysis results
```
