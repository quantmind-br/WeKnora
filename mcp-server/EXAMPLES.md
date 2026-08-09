# WeKnora MCP Server Usage Examples

This document provides detailed usage examples for the WeKnora MCP Server.

## Basic Usage

### 1. Starting the Server

```bash
# Recommended approach - use the main entry point
python main.py

# Check environment configuration
python main.py --check-only

# Enable verbose logging
python main.py --verbose
```

### 2. Environment Configuration Example

```bash
# Set environment variables
export WEKNORA_BASE_URL="http://localhost:8080/api/v1"
export WEKNORA_API_KEY="your_api_key_here"

# Or set them in a .env file
echo "WEKNORA_BASE_URL=http://localhost:8080/api/v1" > .env
echo "WEKNORA_API_KEY=your_api_key_here" >> .env
```

## MCP Tool Usage Examples

Below are usage examples for various MCP tools:

### Tenant Management

#### Create a Tenant
```json
{
  "tool": "create_tenant",
  "arguments": {
    "name": "My Company",
    "description": "Company knowledge management system",
    "business": "technology",
    "retriever_engines": {
      "engines": [
        {"retriever_type": "keywords", "retriever_engine_type": "postgres"},
        {"retriever_type": "vector", "retriever_engine_type": "postgres"}
      ]
    }
  }
}
```

#### List All Tenants
```json
{
  "tool": "list_tenants",
  "arguments": {}
}
```

### Knowledge Base Management

#### Create a Knowledge Base
```json
{
  "tool": "create_knowledge_base",
  "arguments": {
    "name": "Product Documentation Base",
    "description": "Product-related documents and materials",
    "embedding_model_id": "text-embedding-ada-002",
    "summary_model_id": "gpt-3.5-turbo"
  }
}
```

#### List Knowledge Bases
```json
{
  "tool": "list_knowledge_bases",
  "arguments": {}
}
```

#### Get Knowledge Base Details
```json
{
  "tool": "get_knowledge_base",
  "arguments": {
    "kb_id": "kb_123456"
  }
}
```

#### Hybrid Search
```json
{
  "tool": "hybrid_search",
  "arguments": {
    "kb_id": "kb_123456",
    "query": "How to use the API",
    "vector_threshold": 0.7,
    "keyword_threshold": 0.5,
    "match_count": 10
  }
}
```

### Knowledge Management

#### Create Knowledge from a URL
```json
{
  "tool": "create_knowledge_from_url",
  "arguments": {
    "kb_id": "kb_123456",
    "url": "https://docs.example.com/api-guide",
    "enable_multimodel": true
  }
}
```

#### Create Knowledge from Text
```json
{
  "tool": "create_knowledge_from_text",
  "arguments": {
    "kb_id": "my-knowledge-base",
    "title": "Attention Mechanism Summary",
    "content": "# Attention Mechanism\n\nThe attention mechanism allows a model to dynamically allocate weights while processing a sequence...",
    "status": "publish"
  }
}
```

#### List Knowledge
```json
{
  "tool": "list_knowledge",
  "arguments": {
    "kb_id": "kb_123456",
    "page": 1,
    "page_size": 20
  }
}
```

#### Get Knowledge Details
```json
{
  "tool": "get_knowledge",
  "arguments": {
    "knowledge_id": "know_789012"
  }
}
```

### Model Management

#### Create a Model
```json
{
  "tool": "create_model",
  "arguments": {
    "name": "GPT-4 Chat Model",
    "type": "KnowledgeQA",
    "source": "openai",
    "description": "OpenAI GPT-4 model for knowledge Q&A",
    "base_url": "https://api.openai.com/v1",
    "api_key": "sk-...",
    "is_default": true
  }
}
```

#### List Models
```json
{
  "tool": "list_models",
  "arguments": {}
}
```

### Session Management

#### Create a Chat Session
```json
{
  "tool": "create_session",
  "arguments": {
    "kb_id": "kb_123456",
    "max_rounds": 10,
    "enable_rewrite": true,
    "fallback_response": "Sorry, I cannot answer this question.",
    "summary_model_id": "gpt-3.5-turbo"
  }
}
```

#### Get Session Details
```json
{
  "tool": "get_session",
  "arguments": {
    "session_id": "sess_345678"
  }
}
```

#### List Sessions
```json
{
  "tool": "list_sessions",
  "arguments": {
    "page": 1,
    "page_size": 10
  }
}
```

### Chat Functionality

#### Send a Chat Message
```json
{
  "tool": "chat",
  "arguments": {
    "session_id": "sess_345678",
    "query": "Please introduce the main features of the product"
  }
}
```

### Chunk Management

#### List Knowledge Chunks
```json
{
  "tool": "list_chunks",
  "arguments": {
    "knowledge_id": "know_789012",
    "page": 1,
    "page_size": 50
  }
}
```

#### Delete a Knowledge Chunk
```json
{
  "tool": "delete_chunk",
  "arguments": {
    "knowledge_id": "know_789012",
    "chunk_id": "chunk_456789"
  }
}
```

## Complete Workflow Example

### Scenario: Building a Complete Knowledge Q&A System

```bash
# 1. Start the server
python main.py --verbose

# 2. Perform the following steps in the MCP client:
```

#### Step 1: Create a Tenant
```json
{
  "tool": "create_tenant",
  "arguments": {
    "name": "Technical Documentation Center",
    "description": "Company technical documentation knowledge management",
    "business": "technology"
  }
}
```

#### Step 2: Create a Knowledge Base
```json
{
  "tool": "create_knowledge_base",
  "arguments": {
    "name": "API Documentation Base",
    "description": "All API-related documentation"
  }
}
```

#### Step 3: Add Knowledge Content
```json
{
  "tool": "create_knowledge_from_url",
  "arguments": {
    "kb_id": "the returned knowledge base ID",
    "url": "https://docs.company.com/api",
    "enable_multimodel": true
  }
}
```

#### Step 4: Create a Chat Session
```json
{
  "tool": "create_session",
  "arguments": {
    "kb_id": "knowledge base ID",
    "max_rounds": 5,
    "enable_rewrite": true
  }
}
```

#### Step 5: Start the Conversation
```json
{
  "tool": "chat",
  "arguments": {
    "session_id": "session ID",
    "query": "How do I use the user authentication API?"
  }
}
```

## Error Handling Examples

### Common Errors and Solutions

#### 1. Connection Error
```json
{
  "error": "Connection refused",
  "solution": "Check whether WEKNORA_BASE_URL is correct and confirm the service is running"
}
```

#### 2. Authentication Error
```json
{
  "error": "Unauthorized",
  "solution": "Check whether WEKNORA_API_KEY is set correctly"
}
```

#### 3. Resource Not Found
```json
{
  "error": "Knowledge base not found",
  "solution": "Confirm the knowledge base ID is correct, or create a knowledge base first"
}
```

## Advanced Configuration Examples

### Custom Retrieval Configuration
```json
{
  "tool": "hybrid_search",
  "arguments": {
    "kb_id": "kb_123456",
    "query": "search query",
    "vector_threshold": 0.8,
    "keyword_threshold": 0.6,
    "match_count": 15
  }
}
```

### Custom Session Strategy
```json
{
  "tool": "create_session",
  "arguments": {
    "kb_id": "kb_123456",
    "max_rounds": 20,
    "enable_rewrite": true,
    "fallback_response": "Based on the available knowledge, I cannot accurately answer your question. Please try rephrasing it or contact technical support."
  }
}
```

## Performance Optimization Recommendations

1. **Batch Operations**: Batch-process knowledge creation and updates whenever possible
2. **Caching Strategy**: Set search thresholds reasonably to balance accuracy and performance
3. **Session Management**: Promptly clean up unneeded sessions to save resources
4. **Log Monitoring**: Use the `--verbose` option to monitor performance metrics

## Integration Examples

### Integrating with Claude Desktop
Add the following to Claude Desktop's configuration file:
```json
{
  "mcpServers": {
    "weknora": {
      "command": "python",
      "args": ["path/to/main.py"],
      "env": {
        "WEKNORA_BASE_URL": "http://localhost:8080/api/v1",
        "WEKNORA_API_KEY": "your_api_key"
      }
    }
  }
}
```

Project repository: https://github.com/Tencent/WeKnora/tree/main/mcp-server

### Integrating with Other MCP Clients
Refer to each client's documentation to configure the server startup command and environment variables.

## Troubleshooting

If you encounter issues:
1. Run `python main.py --check-only` to check the environment
2. Use `python main.py --verbose` to view detailed logs
3. Check whether the WeKnora service is running normally
4. Verify network connectivity and firewall settings
