# Event System Usage Examples

## Integrating the Event System into the Chat Pipeline

### 1. Set Up the Event Bus During Service Initialization

```go
// internal/container/container.go or main.go

import (
    "github.com/Tencent/WeKnora/internal/event"
)

func InitializeEventSystem() {
    // Get the global event bus
    bus := event.GetGlobalEventBus()
    
    // Register the monitoring handler
    event.NewMonitoringHandler(bus)
    
    // Register the analytics handler
    event.NewAnalyticsHandler(bus)
    
    // Or register a custom handler
    bus.On(event.EventQueryReceived, func(ctx context.Context, e event.Event) error {
        // Custom handling logic
        return nil
    })
}
```

### 2. Emit Events in the Query Processing Service

#### Example: Adding Events in search.go

```go
// internal/application/service/chat_pipline/search.go

import (
    "github.com/Tencent/WeKnora/internal/event"
    "time"
)

func (p *PluginSearch) OnEvent(
    ctx context.Context,
    eventType types.EventType,
    chatManage *types.ChatManage,
    next func() *PluginError,
) *PluginError {
    // Emit the retrieval-start event
    startTime := time.Now()
    event.Emit(ctx, event.NewEvent(event.EventRetrievalStart, event.RetrievalData{
        Query:           chatManage.ProcessedQuery,
        KnowledgeBaseID: chatManage.KnowledgeBaseID,
        TopK:            chatManage.EmbeddingTopK,
        RetrievalType:   "vector",
    }).WithSessionID(chatManage.SessionID))
    
    // Perform the retrieval logic
    results, err := p.performSearch(ctx, chatManage)
    if err != nil {
        // Emit the error event
        event.Emit(ctx, event.NewEvent(event.EventError, event.ErrorData{
            Error:     err.Error(),
            Stage:     "retrieval",
            SessionID: chatManage.SessionID,
            Query:     chatManage.ProcessedQuery,
        }).WithSessionID(chatManage.SessionID))
        return ErrSearch.WithError(err)
    }
    
    // Emit the retrieval-complete event
    event.Emit(ctx, event.NewEvent(event.EventRetrievalComplete, event.RetrievalData{
        Query:           chatManage.ProcessedQuery,
        KnowledgeBaseID: chatManage.KnowledgeBaseID,
        TopK:            chatManage.EmbeddingTopK,
        RetrievalType:   "vector",
        ResultCount:     len(results),
        Duration:        time.Since(startTime).Milliseconds(),
        Results:         results,
    }).WithSessionID(chatManage.SessionID))
    
    chatManage.SearchResult = results
    return next()
}
```

#### Example: Adding Events in rewrite.go

```go
// internal/application/service/chat_pipline/rewrite.go

func (p *PluginRewriteQuery) OnEvent(
    ctx context.Context,
    eventType types.EventType,
    chatManage *types.ChatManage,
    next func() *PluginError,
) *PluginError {
    // Emit the rewrite-start event
    event.Emit(ctx, event.NewEvent(event.EventQueryRewrite, event.QueryData{
        OriginalQuery: chatManage.Query,
        SessionID:     chatManage.SessionID,
    }).WithSessionID(chatManage.SessionID))
    
    // Perform the query rewrite
    rewrittenQuery, err := p.rewriteQuery(ctx, chatManage)
    if err != nil {
        return ErrRewrite.WithError(err)
    }
    
    // Emit the rewrite-complete event
    event.Emit(ctx, event.NewEvent(event.EventQueryRewritten, event.QueryData{
        OriginalQuery:  chatManage.Query,
        RewrittenQuery: rewrittenQuery,
        SessionID:      chatManage.SessionID,
    }).WithSessionID(chatManage.SessionID))
    
    chatManage.RewriteQuery = rewrittenQuery
    return next()
}
```

#### Example: Adding Events in rerank.go

```go
// internal/application/service/chat_pipline/rerank.go

func (p *PluginRerank) OnEvent(
    ctx context.Context,
    eventType types.EventType,
    chatManage *types.ChatManage,
    next func() *PluginError,
) *PluginError {
    // Emit the rerank-start event
    startTime := time.Now()
    inputCount := len(chatManage.SearchResult)
    
    event.Emit(ctx, event.NewEvent(event.EventRerankStart, event.RerankData{
        Query:      chatManage.ProcessedQuery,
        InputCount: inputCount,
        ModelID:    chatManage.RerankModelID,
    }).WithSessionID(chatManage.SessionID))
    
    // Perform the reranking
    rerankResults, err := p.performRerank(ctx, chatManage)
    if err != nil {
        return ErrRerank.WithError(err)
    }
    
    // Emit the rerank-complete event
    event.Emit(ctx, event.NewEvent(event.EventRerankComplete, event.RerankData{
        Query:       chatManage.ProcessedQuery,
        InputCount:  inputCount,
        OutputCount: len(rerankResults),
        ModelID:     chatManage.RerankModelID,
        Duration:    time.Since(startTime).Milliseconds(),
        Results:     rerankResults,
    }).WithSessionID(chatManage.SessionID))
    
    chatManage.RerankResult = rerankResults
    return next()
}
```

#### Example: Adding Events in chat_completion.go

```go
// internal/application/service/chat_pipline/chat_completion.go

func (p *PluginChatCompletion) OnEvent(
    ctx context.Context,
    eventType types.EventType,
    chatManage *types.ChatManage,
    next func() *PluginError,
) *PluginError {
    // Emit the chat-start event
    startTime := time.Now()
    event.Emit(ctx, event.NewEvent(event.EventChatStart, event.ChatData{
        Query:    chatManage.Query,
        ModelID:  chatManage.ChatModelID,
        IsStream: false,
    }).WithSessionID(chatManage.SessionID))
    
    // Prepare the model and messages
    chatModel, opt, err := prepareChatModel(ctx, p.modelService, chatManage)
    if err != nil {
        return ErrGetChatModel.WithError(err)
    }
    
    chatMessages := prepareMessagesWithHistory(chatManage)
    
    // Call the model
    chatResponse, err := chatModel.Chat(ctx, chatMessages, opt)
    if err != nil {
        event.Emit(ctx, event.NewEvent(event.EventError, event.ErrorData{
            Error:     err.Error(),
            Stage:     "chat_completion",
            SessionID: chatManage.SessionID,
            Query:     chatManage.Query,
        }).WithSessionID(chatManage.SessionID))
        return ErrModelCall.WithError(err)
    }
    
    // Emit the chat-complete event
    event.Emit(ctx, event.NewEvent(event.EventChatComplete, event.ChatData{
        Query:      chatManage.Query,
        ModelID:    chatManage.ChatModelID,
        Response:   chatResponse.Content,
        TokenCount: chatResponse.TokenCount,
        Duration:   time.Since(startTime).Milliseconds(),
        IsStream:   false,
    }).WithSessionID(chatManage.SessionID))
    
    chatManage.ChatResponse = chatResponse
    return next()
}
```

### 3. Emit the Request-Received Event at the Handler Layer

```go
// internal/handler/message.go

func (h *MessageHandler) SendMessage(c *gin.Context) {
    ctx := c.Request.Context()
    
    // Parse the request
    var req types.SendMessageRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // Emit the query-received event
    event.Emit(ctx, event.NewEvent(event.EventQueryReceived, event.QueryData{
        OriginalQuery: req.Content,
        SessionID:     req.SessionID,
        UserID:        c.GetString("user_id"),
    }).WithSessionID(req.SessionID).WithRequestID(c.GetString("request_id")))
    
    // Process the message...
}
```

### 4. Custom Monitoring Handler

```go
// internal/monitoring/event_monitor.go

package monitoring

import (
    "context"
    "github.com/Tencent/WeKnora/internal/event"
    "github.com/prometheus/client_golang/prometheus"
)

var (
    retrievalDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "retrieval_duration_milliseconds",
            Help: "Duration of retrieval operations",
        },
        []string{"knowledge_base_id", "retrieval_type"},
    )
    
    rerankDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "rerank_duration_milliseconds",
            Help: "Duration of rerank operations",
        },
        []string{"model_id"},
    )
)

func init() {
    prometheus.MustRegister(retrievalDuration)
    prometheus.MustRegister(rerankDuration)
}

func SetupEventMonitoring() {
    bus := event.GetGlobalEventBus()
    
    // Monitor retrieval performance
    bus.On(event.EventRetrievalComplete, func(ctx context.Context, e event.Event) error {
        data := e.Data.(event.RetrievalData)
        retrievalDuration.WithLabelValues(
            data.KnowledgeBaseID,
            data.RetrievalType,
        ).Observe(float64(data.Duration))
        return nil
    })
    
    // Monitor rerank performance
    bus.On(event.EventRerankComplete, func(ctx context.Context, e event.Event) error {
        data := e.Data.(event.RerankData)
        rerankDuration.WithLabelValues(data.ModelID).Observe(float64(data.Duration))
        return nil
    })
}
```

### 5. Logging Handler

```go
// internal/logging/event_logger.go

package logging

import (
    "context"
    "encoding/json"
    "github.com/Tencent/WeKnora/internal/event"
    "github.com/Tencent/WeKnora/internal/logger"
)

func SetupEventLogging() {
    bus := event.GetGlobalEventBus()
    
    // Perform structured logging for all events
    logHandler := event.ApplyMiddleware(
        func(ctx context.Context, e event.Event) error {
            data, _ := json.Marshal(e.Data)
            logger.Infof(ctx, "Event: type=%s, session=%s, request=%s, data=%s",
                e.Type, e.SessionID, e.RequestID, string(data))
            return nil
        },
        event.WithTiming(),
    )
    
    // Register on all key events
    bus.On(event.EventQueryReceived, logHandler)
    bus.On(event.EventQueryRewritten, logHandler)
    bus.On(event.EventRetrievalComplete, logHandler)
    bus.On(event.EventRerankComplete, logHandler)
    bus.On(event.EventChatComplete, logHandler)
    bus.On(event.EventError, logHandler)
}
```

### 6. Complete Initialization Flow

```go
// cmd/server/main.go or internal/container/container.go

func Initialize() {
    // 1. Initialize the event system
    eventBus := event.GetGlobalEventBus()
    
    // 2. Set up monitoring
    event.NewMonitoringHandler(eventBus)
    
    // 3. Set up analytics
    event.NewAnalyticsHandler(eventBus)
    
    // 4. Set up Prometheus monitoring (if needed)
    // monitoring.SetupEventMonitoring()
    
    // 5. Set up structured logging (if needed)
    // logging.SetupEventLogging()
    
    // 6. Other initialization...
}
```

## Testing the Event System

```go
// Use a dedicated event bus within tests
func TestMyService(t *testing.T) {
    ctx := context.Background()
    
    // Create a test-specific event bus
    testBus := event.NewEventBus()
    
    // Register a test listener
    var receivedEvents []event.Event
    testBus.On(event.EventQueryReceived, func(ctx context.Context, e event.Event) error {
        receivedEvents = append(receivedEvents, e)
        return nil
    })
    
    // Run the test...
    testBus.Emit(ctx, event.NewEvent(event.EventQueryReceived, event.QueryData{
        OriginalQuery: "test",
    }))
    
    // Verify the event
    if len(receivedEvents) != 1 {
        t.Errorf("Expected 1 event, got %d", len(receivedEvents))
    }
}
```

## Asynchronous Processing Example

```go
// For events that don't affect the main flow, an async mode can be used
func SetupAsyncAnalytics() {
    asyncBus := event.NewAsyncEventBus()
    
    asyncBus.On(event.EventQueryReceived, func(ctx context.Context, e event.Event) error {
        // Send to the analytics platform asynchronously, without blocking the main flow
        // sendToAnalyticsPlatform(e)
        return nil
    })
    
    // Emit events using the async bus
    // asyncBus.Emit(ctx, event)
}
```

## Performance Optimization Recommendations

1. **Avoid using the synchronous event bus on the critical path**: For monitoring, logging, and other operations that don't affect business logic, use async mode instead
2. **Use middleware judiciously**: Apply middleware only where needed to avoid unnecessary overhead
3. **Control the size of event data**: Avoid passing large amounts of data in events, especially in async mode
4. **Use dedicated listeners**: Don't do too much within a single listener — keep to a single responsibility
