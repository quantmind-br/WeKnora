# WeKnora MCP Server Installation and Usage Guide

## Quick Start

### 1. Install Dependencies
```bash
pip install -r requirements.txt
```

### 2. Set Environment Variables
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

### 3. Run the Server

There are several ways to run the server:

#### Method 1: Using the Main Entry Point (Recommended)
```bash
python main.py
```

#### Method 2: Using the Original Startup Script
```bash
python run_server.py
```

#### Method 3: Running the Server Module Directly
```bash
python weknora_mcp_server.py
```

#### Method 4: Running as a Python Module
```bash
python -m weknora_mcp_server
```

## Installing as a Python Package

### Development Mode Installation
```bash
pip install -e .
```

Once installed, you can use the command-line tools:
```bash
weknora-mcp-server
# or
weknora-server
```

### Production Mode Installation
```bash
pip install .
```

### Building Distribution Packages
```bash
# Build source distribution and wheel
python setup.py sdist bdist_wheel

# Or use the build tool
pip install build
python -m build
```

## Command-Line Options

The main entry point `main.py` supports the following options:

```bash
python main.py --help                 # Show help information
python main.py --check-only           # Only check environment configuration
python main.py --verbose              # Enable verbose logging
python main.py --version              # Show version information
```

## Environment Check

Run the following command to check your environment configuration:
```bash
python main.py --check-only
```

This will display:
- WeKnora API base URL configuration
- API key setup status
- Dependency package installation status

## Troubleshooting

### 1. Import Errors
If you encounter an `ImportError`, make sure:
- All dependencies are installed: `pip install -r requirements.txt`
- Your Python version is compatible (3.10+ recommended)
- There are no filename conflicts

### 2. Connection Errors
If you cannot connect to the WeKnora API:
- Check that `WEKNORA_BASE_URL` is correct
- Confirm that the WeKnora service is running
- Verify your network connection

### 3. Authentication Errors
If you encounter authentication issues:
- Check that `WEKNORA_API_KEY` is set
- Confirm that the API key is valid
- Verify your permission settings

## Development Mode

### Project Structure
```
WeKnora/mcp-server/
├── __init__.py              # Package initialization file
├── main.py                  # Main entry point
├── run_server.py           # Original startup script
├── weknora_mcp_server.py   # MCP server implementation
├── requirements.txt        # Dependency list
├── setup.py               # Installation script
├── pyproject.toml         # Project metadata (PyPI: tencent-weknora-mcp)
├── MANIFEST.in            # Included files manifest
├── LICENSE                # License
├── README.md              # Project description
└── INSTALL.md             # Installation guide
```

### Adding New Features
1. Add a new API method in the `WeKnoraClient` class
2. Register a new tool function with the `@mcp.tool()` decorator: annotate parameters with types (the schema is auto-generated), write the description in the docstring, and have the function body call the newly added client method above
3. Update documentation and tests

### Testing
```bash
# Run basic tests
python check_imports.py

# Test environment configuration
python main.py --check-only

# Test server startup
python main.py --verbose
```

## Deployment

### Docker Deployment
Create a `Dockerfile`:
```dockerfile
FROM python:3.11-slim

WORKDIR /app
COPY requirements.txt .
RUN pip install -r requirements.txt

COPY . .
RUN pip install -e .

ENV WEKNORA_BASE_URL=http://localhost:8080/api/v1
EXPOSE 8000

CMD ["weknora-mcp-server"]
```

### System Service
Create the systemd service file `/etc/systemd/system/weknora-mcp.service`:
```ini
[Unit]
Description=WeKnora MCP Server
After=network.target

[Service]
Type=simple
User=weknora
WorkingDirectory=/opt/weknora-mcp
Environment=WEKNORA_BASE_URL=http://localhost:8080/api/v1
Environment=WEKNORA_API_KEY=your_api_key
ExecStart=/usr/local/bin/weknora-mcp-server
Restart=always

[Install]
WantedBy=multi-user.target
```

Enable the service:
```bash
sudo systemctl enable weknora-mcp
sudo systemctl start weknora-mcp
```

## Support

If you encounter any issues, please:
1. Check the log output
2. Check the environment configuration
3. Refer to the Troubleshooting section
4. Submit an Issue to the project repository: https://github.com/Tencent/WeKnora/issues
