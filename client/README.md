# WeKnora HTTP Client

This package provides a client library for interacting with the WeKnora service, supporting calls to all HTTP-based interfaces and making it easier for other modules to integrate with the WeKnora service without having to write HTTP request code directly.

## Key Features

The client includes the following main functional modules:

1. **Session management**: create, retrieve, update, and delete sessions
2. **Knowledge base management**: create, retrieve, update, and delete knowledge bases
3. **Knowledge management**: add, retrieve, and delete knowledge content
4. **Space management**: CRUD operations for spaces
5. **Knowledge Q&A**: supports both standard and streaming Q&A
6. **Agent Q&A**: supports Agent-based intelligent Q&A, including the thinking process, tool calls, and reflection
7. **Chunk management**: query, update, and delete knowledge chunks
8. **Message management**: retrieve and delete session messages
9. **Model management**: create, retrieve, update, and delete models

## Usage

### Creating a client instance

```go
import (
    "context"
    "github.com/Tencent/WeKnora/client"
    "time"
)

// Create a client instance
apiClient := client.NewClient(
    "http://api.example.com", 
    client.WithToken("your-auth-token"),
    client.WithTimeout(30*time.Second),
)
```

### Space configuration

The client supports setting a default space via `WithTenantID`, which automatically attaches the `X-Tenant-ID` request header to requests:

```go
tenantID := uint64(10000)
apiClient := client.NewClient(
    "http://api.example.com",
    client.WithToken("your-auth-token"),
    client.WithTenantID(tenantID),
)
```

If a particular request needs to temporarily switch spaces, you can set `TenantID` in the `context`. The value can be `uint64`, `*uint64`, or a number in string form; the client will prioritize this value:

```go
ctx := context.WithValue(context.Background(), "TenantID", uint64(10000))
// Pass ctx when calling any client method to switch to space 10000
```

### Example: Creating a knowledge base and uploading a file

```go
// Create a knowledge base
kb := &client.KnowledgeBase{
    Name:        "Test Knowledge Base",
    Description: "This is a test knowledge base",
    ChunkingConfig: client.ChunkingConfig{
        ChunkSize:    500,
        ChunkOverlap: 50,
        Separators:   []string{"\n\n", "\n", ". ", "? ", "! "},
    },
    ImageProcessingConfig: client.ImageProcessingConfig{
        ModelID: "image_model_id",
    },
    EmbeddingModelID: "embedding_model_id",
    SummaryModelID:   "summary_model_id",
}

kb, err := apiClient.CreateKnowledgeBase(context.Background(), kb)
if err != nil {
    // Handle error
}

// Upload a knowledge file and add metadata
metadata := map[string]string{
    "source": "local",
    "type":   "document",
}
knowledge, err := apiClient.CreateKnowledgeFromFile(context.Background(), kb.ID, "path/to/file.pdf", metadata)
if err != nil {
    // Handle error
}
```

### Example: Creating a session and asking questions

```go
// Create a session
sessionRequest := &client.CreateSessionRequest{
    KnowledgeBaseID: knowledgeBaseID,
    SessionStrategy: &client.SessionStrategy{
        MaxRounds:        10,
        EnableRewrite:    true,
        FallbackStrategy: "fixed_answer",
        FallbackResponse: "Sorry, I can't answer that question",
        EmbeddingTopK:    5,
        KeywordThreshold: 0.5,
        VectorThreshold:  0.7,
        RerankModelID:    "rerank_model_id",
        RerankTopK:       3,
        RerankThreshold:  0.8,
        SummaryModelID:   "summary_model_id",
    },
}

session, err := apiClient.CreateSession(context.Background(), sessionRequest)
if err != nil {
    // Handle error
}

// Standard Q&A
answer, err := apiClient.KnowledgeQA(context.Background(), session.ID, &client.KnowledgeQARequest{
    Query: "What is artificial intelligence?",
})
if err != nil {
    // Handle error
}

// Streaming Q&A
err = apiClient.KnowledgeQAStream(context.Background(), session.ID, &client.KnowledgeQARequest{
    Query:            "What is machine learning?",
    KnowledgeBaseIDs: []string{knowledgeBaseID}, // Optional: specify a knowledge base
    WebSearchEnabled: false,                      // Optional: whether to enable web search
}, func(response *client.StreamResponse) error {
    // Handle each response chunk
    fmt.Print(response.Content)
    return nil
})
if err != nil {
    // Handle error
}
```

### Example: Agent intelligent Q&A

Agent Q&A provides more powerful intelligent conversation capabilities, supporting tool calls, display of the thinking process, and self-reflection.

```go
// Create an Agent session
agentSession := apiClient.NewAgentSession(session.ID)

// Perform Agent Q&A with full event handling
err := agentSession.Ask(context.Background(), "Search for machine learning knowledge and summarize the key points", 
    func(resp *client.AgentStreamResponse) error {
        switch resp.ResponseType {
        case client.AgentResponseTypeThinking:
            // The Agent is thinking
            if resp.Done {
                fmt.Printf("💭 Thinking: %s\n", resp.Content)
            }
        
        case client.AgentResponseTypeToolCall:
            // The Agent calls a tool
            if resp.Data != nil {
                toolName := resp.Data["tool_name"]
                fmt.Printf("🔧 Calling tool: %v\n", toolName)
            }
        
        case client.AgentResponseTypeToolResult:
            // Tool execution result
            fmt.Printf("✓ Tool result: %s\n", resp.Content)
        
        case client.AgentResponseTypeReferences:
            // Knowledge references
            if resp.KnowledgeReferences != nil {
                fmt.Printf("📚 Found %d related knowledge items\n", len(resp.KnowledgeReferences))
                for _, ref := range resp.KnowledgeReferences {
                    fmt.Printf("  - [%.3f] %s\n", ref.Score, ref.KnowledgeTitle)
                }
            }
        
        case client.AgentResponseTypeAnswer:
            // Final answer (streamed)
            fmt.Print(resp.Content)
            if resp.Done {
                fmt.Println() // Newline once finished
            }
        
        case client.AgentResponseTypeReflection:
            // The Agent's self-reflection
            if resp.Done {
                fmt.Printf("🤔 Reflection: %s\n", resp.Content)
            }
        
        case client.AgentResponseTypeError:
            // Error information
            fmt.Printf("❌ Error: %s\n", resp.Content)
        }
        return nil
    })

if err != nil {
    // Handle error
}

// Simplified version: only care about the final answer
var finalAnswer string
err = agentSession.Ask(context.Background(), "What is deep learning?", 
    func(resp *client.AgentStreamResponse) error {
        if resp.ResponseType == client.AgentResponseTypeAnswer {
            finalAnswer += resp.Content
        }
        return nil
    })
```

### Agent event type reference

| Event Type | Description | When Triggered |
|---------|------|---------|
| `AgentResponseTypeThinking` | Agent thinking process | When the Agent is analyzing the problem and formulating a plan |
| `AgentResponseTypeToolCall` | Tool call | When the Agent decides to use a tool |
| `AgentResponseTypeToolResult` | Tool execution result | After tool execution completes |
| `AgentResponseTypeReferences` | Knowledge references | When related knowledge is retrieved |
| `AgentResponseTypeAnswer` | Final answer | When the Agent generates a response (streamed) |
| `AgentResponseTypeReflection` | Self-reflection | When the Agent evaluates its own answer |
| `AgentResponseTypeError` | Error | When an error occurs |

### Agent Q&A testing tool

We provide an interactive command-line tool for testing Agent functionality:

```bash
cd client/cmd/agent_test
go build -o agent_test
./agent_test -url http://localhost:8080 -kb <knowledge_base_id>
```

This tool supports:
- Creating and managing sessions
- Interactive Agent Q&A
- Real-time display of all Agent events
- Performance statistics and debugging information

For detailed usage instructions, please refer to `client/cmd/agent_test/README.md`.

### Advanced usage of Agent Q&A

For more advanced usage examples, please refer to the `agent_example.go` file, which includes:
- Basic Agent Q&A
- Tool call tracking
- Knowledge reference capture
- Full event tracing
- Custom error handling
- Stream cancellation control
- Multi-session management

```

### Example: Managing models

```go
// Create a model
modelRequest := &client.CreateModelRequest{
    Name:        "Test Model",
    Type:        client.ModelTypeChat,
    Source:      client.ModelSourceInternal,
    Description: "This is a test model",
    Parameters: client.ModelParameters{
        "temperature": 0.7,
        "top_p":       0.9,
    },
    IsDefault: true,
}
model, err := apiClient.CreateModel(context.Background(), modelRequest)
if err != nil {
    // Handle error
}

// List all models
models, err := apiClient.ListModels(context.Background())
if err != nil {
    // Handle error
}
```

### Example: Managing knowledge chunks

```go
// List knowledge chunks
chunks, total, err := apiClient.ListKnowledgeChunks(context.Background(), knowledgeID, 1, 10)
if err != nil {
    // Handle error
}

// Update a chunk
updateRequest := &client.UpdateChunkRequest{
    Content:   "Updated chunk content",
    IsEnabled: true,
}
updatedChunk, err := apiClient.UpdateChunk(context.Background(), knowledgeID, chunkID, updateRequest)
if err != nil {
    // Handle error
}
```

### Example: Re-parsing knowledge

```go
// Re-parse knowledge (deletes existing content and re-parses)
// Applicable scenarios:
// 1. The original parsing failed and needs to be retried
// 2. The parsing configuration was updated (e.g. chunking strategy, multimodal settings, etc.) and needs to be re-parsed
// 3. The knowledge content has been updated and the parsing results need to be refreshed

knowledge, err := apiClient.ReparseKnowledge(context.Background(), knowledgeID)
if err != nil {
    // Handle error
}

// The knowledge will enter the "pending" state and be re-parsed asynchronously
fmt.Printf("Knowledge ID: %s\n", knowledge.ID)
fmt.Printf("Parse Status: %s\n", knowledge.ParseStatus)      // "pending"
fmt.Printf("Enable Status: %s\n", knowledge.EnableStatus)    // "disabled"

// You can poll to check the parsing status
for {
    time.Sleep(5 * time.Second)
    knowledge, err := apiClient.GetKnowledge(context.Background(), knowledgeID)
    if err != nil {
        // Handle error
    }
    
    if knowledge.ParseStatus == "completed" {
        fmt.Println("Knowledge re-parsing completed!")
        break
    } else if knowledge.ParseStatus == "failed" {
        fmt.Printf("Knowledge re-parsing failed: %s\n", knowledge.ErrorMessage)
        break
    }
}
```

### Example: Canceling parsing

```go
// Cancel an in-progress parsing task (useful when resources are constrained or the wrong file was uploaded)
// - Knowledge that has already reached completed / failed cannot be canceled
// - Chunks/indexes already written will be retained, and ReparseKnowledge can be called later to re-parse

knowledge, err := apiClient.CancelKnowledgeParse(context.Background(), knowledgeID)
if err != nil {
    // Handle error
}
fmt.Printf("Parse Status: %s\n", knowledge.ParseStatus) // "cancelled"
```

### Example: Viewing document parsing traces (Span tree)

```go
// Get the Span tree for the document parsing pipeline (root → stage → subspan)
// - passing 0 for attempt retrieves the latest parsing attempt
// - always returns 5 standard stages: docreader / chunking / embedding / multimodal / postprocess
trace, err := apiClient.GetKnowledgeProcessingSpans(context.Background(), knowledgeID, 0)
if err != nil {
    // Handle error
}
fmt.Printf("ParseStatus=%s CurrentStage=%s\n", trace.ParseStatus, trace.CurrentStage)
for _, stage := range trace.Trace.Children {
    fmt.Printf("- %s: %s (%dms)\n", stage.Name, stage.Status, stage.DurationMs)
}
```

### Example: Retrieving session messages

```go
// Get recent messages
messages, err := apiClient.GetRecentMessages(context.Background(), sessionID, 10)
if err != nil {
    // Handle error
}

// Get messages before a specified time
beforeTime := time.Now().Add(-24 * time.Hour)
olderMessages, err := apiClient.GetMessagesBefore(context.Background(), sessionID, beforeTime, 10)
if err != nil {
    // Handle error
}
```

## Complete Example

Please refer to the `ExampleUsage` function in the `example.go` file, which demonstrates the complete usage workflow of the client.
