# WeKnora MCP Server

This is a Model Context Protocol (MCP) server that provides access to the WeKnora knowledge management API.

## Quick Start

> We recommend referring directly to the [MCP Configuration Guide](./MCP_CONFIG.md), which makes the steps below unnecessary.

### 1. Install dependencies
```bash
pip install -r requirements.txt
```

### 2. Configure environment variables
```bash
# Linux/macOS
export WEKNORA_BASE_URL="http://localhost:8080/api/v1"
export WEKNORA_API_KEY="your_api_key_here"

# Windows PowerShell
$env:WEKNORA_BASE_URL="http://localhost:8080/api/v1"
$env:WEKNORA_API_KEY="your_api_key_here"

# Windows CMD
set WEKNORA_BASE_URL=http://localhost:8080/api/v1
set WEKNORA_API_KEY=your_api_key_here
```

### 3. Run the server

**Recommended approach — using the main entry point:**
```bash
python main.py
```

**Other ways to run it:**
```bash
# Using the original startup script
python run_server.py

# Using the convenience script
python run.py

# Running the server module directly
python weknora_mcp_server.py

# Running as a Python module
python -m weknora_mcp_server
```

### 4. Command-line options
```bash
python main.py --help                 # Show help information
python main.py --check-only           # Only check environment configuration
python main.py --verbose              # Enable verbose logging
python main.py --version              # Show version information
```

## Installing as a Python package

### Installing from PyPI

```bash
pip install tencent-weknora-mcp
# Or run directly with uvx (no pre-installation required)
uvx --from tencent-weknora-mcp weknora-mcp-server
```

> The official PyPI package name is **`tencent-weknora-mcp`** (maintained by Tencent/WeKnora, published via Trusted Publishing).
> Please stop using the old community package `weknora-mcp`.
> After installation, the command-line entry points remain `weknora-mcp-server` / `weknora-server`.

### Installing in development mode
```bash
pip install -e .
```

Once installed, you can use the command-line tools:
```bash
weknora-mcp-server
# or
weknora-server
```

### Installing in production mode
```bash
pip install .
```

### Building distribution packages
```bash
# Using setuptools
python setup.py sdist bdist_wheel

# Using modern build tools
pip install build
python -m build
```

## Testing the module

Run the test script to verify the module is working correctly:
```bash
python test_module.py
```

## Features

This MCP server provides the following tools:

### Space management
- `create_tenant` - Create a new space
- `list_tenants` - List all spaces

### Knowledge base management
- `create_knowledge_base` - Create a knowledge base
- `list_knowledge_bases` - List knowledge bases
- `get_knowledge_base` - Get knowledge base details
- `delete_knowledge_base` - Delete a knowledge base
- `hybrid_search` - Hybrid search

### Knowledge management
- `create_knowledge_from_file` - Create knowledge from a local file
- `create_knowledge_from_url` - Create knowledge from a URL
- `create_knowledge_from_text` - Create knowledge from text
- `list_knowledge` - List knowledge
- `get_knowledge` - Get knowledge details
- `delete_knowledge` - Delete knowledge

### Model management
- `create_model` - Create a model
- `list_models` - List models
- `get_model` - Get model details

### Session management
- `create_session` - Create a chat session
- `get_session` - Get session details
- `list_sessions` - List sessions
- `delete_session` - Delete a session

### Chat functionality
- `chat` - Send a chat message

### Chunk management
- `list_chunks` - List knowledge chunks
- `delete_chunk` - Delete a knowledge chunk

## Troubleshooting

If you encounter import errors, please make sure that:
1. All required dependency packages are installed
2. Your Python version is compatible (3.10+ recommended)
3. There are no filename conflicts (avoid using `mcp.py` as a filename)

## Demo

<img width="950" height="2063" alt="118d078426f42f3d4983c13386085d7f" src="https://github.com/user-attachments/assets/09111ec8-0489-415c-969d-aa3835778e14" />
