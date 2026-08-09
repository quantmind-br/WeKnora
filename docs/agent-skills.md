Vou traduzir o documento diretamente, preservando toda a estrutura markdown.

--- DOCUMENT START ---
# Agent Skills Documentation

## Overview

Agent Skills is an extension mechanism that lets an Agent learn new capabilities by reading an "instruction manual." Unlike traditional hard-coded tools, Skills extend the Agent's capabilities by injecting content into the System Prompt, following the **Progressive Disclosure** design philosophy.
Currently only available to agents with **intelligent reasoning** capability. The relevant configuration can be found on the agent's edit page in the frontend.

### Core Features

- **Non-intrusive extension**: Does not affect the original Agent ReAct workflow
- **On-demand loading**: Three-tier progressive loading to optimize token usage
- **Sandboxed execution**: Scripts run securely in an isolated environment
- **Flexible configuration**: Supports multiple directories and allowlist filtering

## Design Philosophy

### Progressive Disclosure

Skills use a three-tier loading mechanism to ensure detailed information is only provided to the LLM when needed:

```
┌─────────────────────────────────────────────────────────────────┐
│ Level 1: Metadata                                                │
│ • Always loaded into the System Prompt                           │
│ • About 100 tokens/skill                                         │
│ • Contains: skill name + short description                       │
└─────────────────────────────────────────────────────────────────┘
                              ↓ When a user request matches
┌─────────────────────────────────────────────────────────────────┐
│ Level 2: Instructions                                             │
│ • Loaded on demand via the read_skill tool                       │
│ • The instruction content of SKILL.md                            │
│ • Contains: detailed instructions, code examples, usage          │
└─────────────────────────────────────────────────────────────────┘
                              ↓ When more information is needed
┌─────────────────────────────────────────────────────────────────┐
│ Level 3: Resources                                                │
│ • Specific files loaded via the read_skill tool                  │
│ • Supplementary docs, config templates, script files             │
│ • Scripts executed via execute_skill_script                      │
└─────────────────────────────────────────────────────────────────┘
```

## Skill Directory Structure

Each Skill is a directory containing a main `SKILL.md` file and optional additional resources:

```
my-skill/
├── SKILL.md           # Required: main file (includes YAML frontmatter)
├── REFERENCE.md       # Optional: supplementary documentation
├── templates/         # Optional: template files
│   └── config.yaml
└── scripts/           # Optional: executable scripts
    ├── analyze.py
    └── generate.sh
```

## SKILL.md Format

### YAML Frontmatter

Every `SKILL.md` must start with YAML frontmatter defining its metadata:

```markdown
---
name: pdf-processing
description: Extract text and tables from PDF files, fill forms, merge documents. Use when working with PDF files or when the user mentions PDFs, forms, or document extraction.
---

# PDF Processing

This skill provides utilities for working with PDF documents.

## Quick Start

Use pdfplumber to extract text from PDFs:

```python
import pdfplumber

with pdfplumber.open("document.pdf") as pdf:
    text = pdf.pages[0].extract_text()
    print(text)
```

## Metadata Validation Rules

| Field | Requirement |
|------|------|
| `name` | 1–50 characters, only Chinese characters, English letters, and digits allowed, cannot be a reserved word |
| `description` | 1–500 characters, describing the skill's purpose and trigger conditions |

**Reserved words**: `system`, `default`, `internal`, `core`, `base`, `root`, `admin`


## Configuration

### AgentConfig Options

```go
type AgentConfig struct {
    // ... other configuration ...

    // Skills-related configuration
    SkillsEnabled  bool     `json:"skills_enabled"`   // Whether Skills are enabled
    SkillDirs      []string `json:"skill_dirs"`       // List of Skill directories
    AllowedSkills  []string `json:"allowed_skills"`   // Allowlist (empty = all allowed)
}
```

### Configuration Example

```json
{
  "skills_enabled": true,
  "skill_dirs": [
    "/path/to/project/skills",
    "/home/user/.agent-skills"
  ],
  "allowed_skills": ["pdf-processing", "code-review"]
}
```

### Sandbox Configuration (Environment Variables)

Sandbox-related configuration is set via environment variables:

| Environment Variable | Description | Default |
|---------|------|--------|
| `WEKNORA_SANDBOX_MODE` | Sandbox mode: `docker`, `local`, `disabled` | `disabled` |
| `WEKNORA_SANDBOX_TIMEOUT` | Script execution timeout (seconds) | `60` |
| `WEKNORA_SANDBOX_DOCKER_IMAGE` | Custom Docker image | `wechatopenai/weknora-sandbox:latest` |

### Sandbox Modes

| Mode | Description |
|------|------|
| `docker` | Uses Docker container isolation (recommended) |
| `local` | Local process execution (basic security restrictions) |
| `disabled` | Disables script execution |

## Agent Tools

The Skills feature interacts with the Agent via two tools:

### read_skill

Reads skill content or a specific file.

**Parameters**:
```json
{
  "skill_name": "pdf-processing",      // Required: skill name
  "file_path": "FORMS.md"              // Optional: relative path
}
```

**Use cases**:
1. Loading Level 2 content: pass only `skill_name`
2. Loading Level 3 resources: pass both `skill_name` and `file_path`

**Example calls**:
```json
// Load the skill's main content
{"skill_name": "pdf-processing"}

// Load supplementary documentation
{"skill_name": "pdf-processing", "file_path": "FORMS.md"}

// View script content
{"skill_name": "pdf-processing", "file_path": "scripts/analyze.py"}
```

### execute_skill_script

Executes a skill script in the sandbox.

**Parameters**:
```json
{
  "skill_name": "pdf-processing",           // Required: skill name
  "script_path": "scripts/analyze.py",      // Required: relative script path
  "args": ["input.pdf", "--format", "json"] // Optional: command-line arguments
}
```

**Supported script types**:
- Python (`.py`)
- Shell (`.sh`)
- JavaScript/Node.js (`.js`)
- Ruby (`.rb`)
- Go (`.go`)

## Preloaded Skills

The system ships with the following 5 built-in preloaded skills, used to enhance knowledge base Q&A and document processing capabilities:

### 1. citation-generator - Citation Generator

**Purpose**: Automatically generates standardized citation formats

**Trigger scenarios**:
- Need to generate references
- Annotating the source of knowledge base content
- Requests to provide citation information

**Core capabilities**:
| Feature | Description |
|------|------|
| Source annotation | Annotates the source for each piece of knowledge used in an answer |
| Formatted citations | Supports APA, MLA, Chicago, and simplified formats |
| Reference list | Generates a complete reference list at the end of the answer |

**Simplified citation format example**:
```
According to company policy [Employee Handbook 2024.pdf, page 15], annual leave requests must be submitted in advance...
```

---

### 2. data-processor - Data Processor

**Purpose**: Data processing and analysis

**Trigger scenarios**:
- "Analyze this data," "run the stats," "calculate the total/average"
- "Convert to JSON/CSV format"
- "Extract key information," "organize into a table"
- "Generate a report," "summarize the data"

**Core capabilities**:
| Feature | Description |
|------|------|
| Data analysis | Performs statistical analysis on retrieved document data |
| Format conversion | Converts between JSON/CSV/Markdown and other formats |
| Data extraction | Extracts structured information from unstructured text |
| Report generation | Generates data analysis reports and summaries |

**Available scripts**:
- `scripts/analyze.py` - Data analysis script
- `scripts/format_converter.py` - Format conversion script
- `scripts/extract_info.py` - Information extraction script

**Script usage examples**:
```bash
# Data analysis
echo '{"items": [1, 2, 3, 4, 5]}' | python scripts/analyze.py

# Format conversion (JSON to CSV)
echo '[{"name": "A", "value": 1}]' | python scripts/format_converter.py --to csv

# Information extraction
echo "2024 sales revenue was 1 million yuan" | python scripts/extract_info.py
```

---

### 3. doc-coauthoring - Document Co-authoring (adapted from Claude's official Skill)

**Purpose**: Guides users through structured document creation

**Trigger scenarios**:
- Writing a document: "write a doc," "draft a proposal," "create a spec"
- Document types: PRD, design doc, decision doc, RFC

**Workflow**:

```
Stage 1: Context Gathering
        ↓
Stage 2: Refinement & Structure
        ↓
Stage 3: Reader Testing
```

**Three-stage overview**:
| Stage | Goal | Key activities |
|------|------|----------|
| Stage 1 | Narrow the information gap between the user and Claude | Meta-information questions, context gathering, clarifying questions |
| Stage 2 | Build the document section by section | Brainstorming, filtering and organizing, iterative revision |
| Stage 3 | Test the document's effectiveness for readers | Predicting reader questions, sub-agent testing, fixing blind spots |

---

### 4. document-analyzer - Document Analyzer

**Purpose**: Deep analysis of document structure and content

**Trigger scenarios**:
- Analyzing document structure
- Extracting key information
- Identifying document type
- Assessing content quality

**Core capabilities**:
| Feature | Description |
|------|------|
| Structure analysis | Identifies the document's section hierarchy and organization |
| Key information extraction | Extracts core arguments, key data, and important conclusions |
| Document type identification | Determines the document type (report, manual, paper, contract, etc.) |
| Content quality assessment | Evaluates the document's completeness, consistency, and readability |

**Analysis workflow**:
1. **Document overview** - Get basic document information
2. **Structure analysis** - Identify heading hierarchy and section organization
3. **Content extraction** - Extract core themes, key arguments, and supporting data
4. **Quality assessment** - Evaluate completeness, consistency, and clarity

---

### Skill Directory Structure

Preloaded skills are located under the `skills/preloaded/` directory:

```
skills/preloaded/
├── citation-generator/
│   └── SKILL.md
├── data-processor/
│   ├── SKILL.md
│   └── scripts/
│       ├── analyze.py
│       ├── format_converter.py
│       └── extract_info.py
├── doc-coauthoring/
│   └── SKILL.md
├── document-analyzer/
│   └── SKILL.md
└── summary-generator/
    └── SKILL.md
```

## Creating a Custom Skill

User-created custom Skills are not yet supported.


## Sandbox Security Mechanisms

### Script Security Validation (Script Validator)

Before a script executes, the system performs multi-layer security validation to intercept potentially malicious operations:

#### Validation Types

| Type | Description | Example |
|------|------|------|
| **Dangerous command detection** | Detects commands that could damage the system | `rm -rf /`, `mkfs`, `shutdown`, fork bombs |
| **Dangerous pattern matching** | Regex matching against high-risk operation patterns | `curl \| bash`, `base64 -d`, `eval()` |
| **Network access detection** | Detects attempted network requests | `curl`, `wget`, `socket.connect`, `requests.get` |
| **Reverse shell detection** | Detects remote-control backdoors | `/dev/tcp/`, `bash -i`, `nc -e` |
| **Argument injection detection** | Detects injection in command-line arguments | `&&`, `\|`, `$()`, backticks |
| **Stdin injection detection** | Detects embedded commands in standard input | Embedded command substitution syntax |

#### Intercepted Dangerous Commands

**System-destructive**:
- `rm -rf /`, `rm -rf /*` - Recursive deletion of the root directory
- `mkfs`, `dd if=/dev/zero` - Filesystem/disk operations
- Fork bombs: `:(){ :|:& };:`

**System control**:
- `shutdown`, `reboot`, `halt`, `poweroff`
- `killall`, `pkill`
- `systemctl`, `service`

**Privilege escalation**:
- `chmod 777 /`, `chown root`
- `setuid`, `setgid`, `passwd`
- Accessing `/etc/passwd`, `/etc/shadow`, `/etc/sudoers`

**Credential theft**:
- Accessing `.ssh/`, `id_rsa`, `id_ed25519`
- Reading sensitive configuration files

**Container escape**:
- `docker`, `kubectl`, `nsenter`
- `unshare`, `capsh`

#### Intercepted Dangerous Patterns

**Code injection**:
```
# The following patterns are intercepted
curl ... | bash           # Download and execute
wget ... | sh              # Download and execute
eval()                     # Dynamic code execution
exec()                     # Command execution
os.system()                # System command execution
subprocess.Popen(shell=True)  # Shell command execution
```

**Encoding bypass attempts**:
```
# The following patterns are intercepted
base64 -d                  # Base64 decode and execute
echo ... | base64 -d       # Piped decode
xxd -r                     # Hex decode
```

**Python-specific risks**:
```python
# The following patterns are intercepted
__import__()               # Dynamic import
pickle.load()               # Deserialization (can execute arbitrary code)
yaml.load()                 # Unsafe YAML loading
yaml.unsafe_load()          # Explicit unsafe loading
```

#### Shell Operator Interception

The following operators in arguments are intercepted:

| Operator | Description |
|--------|------|
| `&&`, `\|\|` | Command chaining |
| `;` | Command separator |
| `\|` | Pipe |
| `$()`, `` ` `` | Command substitution |
| `>`, `>>`, `<` | Redirection |
| `2>`, `&>` | Error/combined redirection |
| `\n`, `\r` | Newline injection |

#### Validation Results

When validation fails, detailed error information is returned:

```go
type ValidationError struct {
    Type    string // Error type: dangerous_command, dangerous_pattern, arg_injection, etc.
    Pattern string // Matched pattern
    Context string // Context information
    Message string // Human-readable description
}
```

**Example error**:
```
security validation failed [dangerous_command]: Script contains dangerous command: rm -rf / (pattern: rm -rf /, context: ...cleanup && rm -rf / && echo done...)
```

#### Usage Example

```go
// Create validator
validator := sandbox.NewScriptValidator()

// Validate script content
result := validator.ValidateScript(scriptContent)
if !result.Valid {
    for _, err := range result.Errors {
        log.Printf("Security error: %s", err.Error())
    }
    return errors.New("script validation failed")
}

// Validate command-line arguments
argsResult := validator.ValidateArgs(args)

// Validate standard input
stdinResult := validator.ValidateStdin(stdin)

// Or validate everything at once
fullResult := validator.ValidateAll(scriptContent, args, stdin)
```

---

### Docker Sandbox

Docker mode provides the strongest isolation:

- **Non-root user**: Runs as a regular user inside the container
- **Capability restrictions**: All Linux capabilities are removed
- **Read-only filesystem**: The root filesystem is read-only
- **Resource limits**: 256MB memory, CPU limits
- **Network isolation**: No network access by default
- **Temporary mounts**: The Skill directory is mounted read-only
- **Script pre-validation**: Security validation runs before execution

#### Sandbox Image

The system uses a dedicated sandbox image, `wechatopenai/weknora-sandbox`, pre-installed with Python 3.11, Node.js 20, common CLI tools, and Python libraries — no need to install dependencies at execution time.

**Pre-pulling the image** (recommended during initial deployment to avoid waiting for a download on the first script execution):

```bash
# Method 1: Pull directly
docker pull wechatopenai/weknora-sandbox:latest

# Method 2: Build locally
sh scripts/build_images.sh -s
```

> If the image is not pre-pulled, the application will automatically and asynchronously pull it on startup (`EnsureImage`), but the first execution may need to wait for the download to complete.

**Built-in image environment**:
- Python 3.11 + pip (requests, pyyaml, pandas, beautifulsoup4)
- Node.js 20 + npm
- CLI tools: jq, curl, bash, grep, sed, awk, etc.

```bash
# Docker execution example
docker run --rm \
  --user 1000:1000 \
  --cap-drop ALL \
  --read-only \
  --memory=256m \
  --network=none \
  -v /path/to/skill:/skill:ro \
  -w /skill \
  wechatopenai/weknora-sandbox:latest \
  python scripts/analyze.py input.pdf
```

### Local Sandbox

Local mode provides basic protection:

- **Command allowlist**: Only specific interpreters are permitted
- **Working directory restriction**: Confined to the Skill directory
- **Environment variable filtering**: Only safe variables are passed through
- **Timeout control**: 30-second default timeout
- **Path traversal protection**: Prevents access to files outside the Skill directory
- **Script pre-validation**: Security validation runs before execution

**Allowed commands**:
- `python`, `python3`
- `node`, `nodejs`
- `bash`, `sh`
- `ruby`
- `go run`

## API Reference

### SkillManager

```go
type Manager interface {
    // Initialize, discovering all Skills
    Initialize(ctx context.Context) error
    
    // Get metadata for all Skills (Level 1)
    GetAllMetadata() []*SkillMetadata
    
    // Load Skill instructions (Level 2)
    LoadSkill(ctx context.Context, skillName string) (*Skill, error)
    
    // Read Skill file content (Level 3)
    ReadSkillFile(ctx context.Context, skillName, filePath string) (string, error)
    
    // List all files in a Skill
    ListSkillFiles(ctx context.Context, skillName string) ([]string, error)
    
    // Execute a Skill script
    ExecuteScript(ctx context.Context, skillName, scriptPath string, args []string) (*sandbox.ExecuteResult, error)
    
    // Check whether it's enabled
    IsEnabled() bool
}
```

### Skill Struct

```go
type Skill struct {
    Name         string // Skill name
    Description  string // Skill description
    BasePath     string // Absolute directory path
    FilePath     string // Absolute path to SKILL.md
    Instructions string // Main instruction content of SKILL.md
    Loaded       bool   // Whether Level 2 has been loaded
}

type SkillMetadata struct {
    Name        string // Skill name
    Description string // Skill description
    BasePath    string // Directory path
}
```

### ExecuteResult Struct

```go
type ExecuteResult struct {
    ExitCode int           // Exit code
    Stdout   string        // Standard output
    Stderr   string        // Standard error
    Duration time.Duration // Execution duration
    Error    error         // Execution error
}
```

## Example: Complete Workflow

Below is the complete process by which an Agent handles a user request:

```
User: "Help me extract table data from report.pdf"

Agent thinking:
  → Looks at the Skills list in the System Prompt
  → Finds that the "pdf-processing" skill matches

Agent action 1: Calls read_skill
  → {"skill_name": "pdf-processing"}
  → Retrieves the SKILL.md instruction content
  → Learns how to use pdfplumber

Agent action 2: Calls execute_skill_script
  → {"skill_name": "pdf-processing", 
     "script_path": "scripts/extract_text.py",
     "args": ["report.pdf"]}
  → The script runs in the sandbox and returns the extracted table data

Agent reply:
  → Presents the extracted table data to the user
  → Offers suggestions for using the data
```

## Troubleshooting

### Skill Not Discovered

1. Check that the `skill_dirs` configuration is correct
2. Confirm that a `SKILL.md` file exists in the directory
3. Verify the YAML frontmatter format

```bash
# Run the demo to verify
go run ./cmd/skills-demo/main.go
```

### Script Execution Failure

1. Check the `sandbox_mode` configuration
2. Docker mode: confirm the Docker service is running
3. Local mode: confirm the interpreter is installed
4. Check script permissions and syntax

### Metadata Validation Errors

Common errors:
- `skill name too long`: Name exceeds 50 characters
- `skill name contains invalid characters`: Contains illegal characters
- `skill name is reserved`: Uses a reserved word
- `skill description too long`: Description exceeds 500 characters
--- DOCUMENT END ---
