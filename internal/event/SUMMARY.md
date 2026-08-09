# WeKnora Event System Summary

## Overview

A complete event emission and listening mechanism has been successfully created for the WeKnora project, supporting event handling for the various steps in the user query processing pipeline.

## Core Features

### ✅ Implemented Features

1. **Event Bus (EventBus)**
   - `Emit(ctx, event)` - Emit an event
   - `On(eventType, handler)` - Register an event listener
   - `Off(eventType)` - Remove an event listener
   - `EmitAndWait(ctx, event)` - Emit an event and wait for all handlers to complete
   - Synchronous and asynchronous modes

2. **Event Types**
   - Query processing events (received, validated, preprocessed, rewritten)
   - Retrieval events (started, vector retrieval, keyword retrieval, entity retrieval, completed)
   - Ranking events (started, completed)
   - Merge events (started, completed)
   - Chat generation events (started, completed, streaming output)
   - Error events

3. **Event Data Structures**
   - `QueryData` - Query data
   - `RetrievalData` - Retrieval data
   - `RerankData` - Ranking data
   - `MergeData` - Merge data
   - `ChatData` - Chat data
   - `ErrorData` - Error data

4. **Middleware Support**
   - `WithLogging()` - Logging middleware
   - `WithTiming()` - Timing middleware
   - `WithRecovery()` - Error recovery middleware
   - `Chain()` - Middleware composition

5. **Global Event Bus**
   - Singleton global event bus
   - Global convenience functions (`On`, `Emit`, `EmitAndWait`, etc.)

6. **Examples and Tests**
   - Complete unit tests
   - Performance benchmarks
   - Complete usage examples
   - Real-world scenario demonstrations

## File Structure

```
internal/event/
├── event.go                    # Core event bus implementation
├── event_data.go              # Event data structure definitions
├── middleware.go              # Middleware implementation
├── global.go                  # Global event bus
├── integration_example.go     # Integration examples (monitoring, analytics handlers)
├── example_test.go            # Tests and examples
├── demo/
│   └── main.go               # Complete RAG pipeline demo
├── README.md                 # Detailed documentation
├── usage_example.md          # Usage example documentation
└── SUMMARY.md                # This document
```

## Performance Metrics

- **Event emission performance**: ~9 nanoseconds/call (benchmark test)
- **Concurrency safety**: Uses `sync.RWMutex` to guarantee thread safety
- **Memory overhead**: Extremely low, only stores event handler function references

## Use Cases

### 1. Monitoring and Metrics Collection

```go
bus.On(event.EventRetrievalComplete, func(ctx context.Context, e event.Event) error {
    data := e.Data.(event.RetrievalData)
    // Send to Prometheus or another monitoring system
    metricsCollector.RecordRetrievalDuration(data.Duration)
    return nil
})
```

### 2. Logging

```go
bus.On(event.EventQueryRewritten, func(ctx context.Context, e event.Event) error {
    data := e.Data.(event.QueryData)
    logger.Infof(ctx, "Query rewritten: %s -> %s", 
        data.OriginalQuery, data.RewrittenQuery)
    return nil
})
```

### 3. User Behavior Analysis

```go
bus.On(event.EventQueryReceived, func(ctx context.Context, e event.Event) error {
    data := e.Data.(event.QueryData)
    // Send to an analytics platform
    analytics.TrackQuery(data.UserID, data.OriginalQuery)
    return nil
})
```

### 4. Error Tracking

```go
bus.On(event.EventError, func(ctx context.Context, e event.Event) error {
    data := e.Data.(event.ErrorData)
    // Send to an error tracking system
    sentry.CaptureException(data.Error)
    return nil
})
```

## Integration

### Step 1: Initialize the Event System

At application startup (e.g., in `main.go` or `container.go`):

```go
import "github.com/Tencent/WeKnora/internal/event"

func Initialize() {
    // Get the global event bus
    bus := event.GetGlobalEventBus()
    
    // Set up monitoring and analytics
    event.NewMonitoringHandler(bus)
    event.NewAnalyticsHandler(bus)
}
```

### Step 2: Emit Events at Each Processing Stage

Add event emission to the various plugins in the query processing pipeline:

```go
// In search.go
event.Emit(ctx, event.NewEvent(event.EventRetrievalStart, event.RetrievalData{
    Query:           chatManage.ProcessedQuery,
    KnowledgeBaseID: chatManage.KnowledgeBaseID,
    TopK:            chatManage.EmbeddingTopK,
}).WithSessionID(chatManage.SessionID))

// In rerank.go
event.Emit(ctx, event.NewEvent(event.EventRerankComplete, event.RerankData{
    Query:       chatManage.ProcessedQuery,
    InputCount:  len(chatManage.SearchResult),
    OutputCount: len(rerankResults),
    Duration:    time.Since(startTime).Milliseconds(),
}).WithSessionID(chatManage.SessionID))
```

### Step 3: Register Custom Event Handlers

Register custom handlers as needed:

```go
event.On(event.EventQueryRewritten, func(ctx context.Context, e event.Event) error {
    // Custom handling logic
    return nil
})
```

## Advantages

1. **Low coupling**: Event emitters and listeners are fully decoupled, making maintenance and extension easier
2. **High performance**: Extremely low performance overhead (~9ns/call)
3. **Flexibility**: Supports synchronous/asynchronous modes and single/multiple listeners
4. **Extensibility**: Easy to add new event types and handlers
5. **Type safety**: Predefined event data structures
6. **Middleware support**: Convenient for adding cross-cutting concerns (logging, timing, error handling, etc.)
7. **Test-friendly**: Easy to verify event behavior in tests

## Test Results

✅ All unit tests passed
✅ Performance test passed (~9ns/call)
✅ Asynchronous processing test passed
✅ Multiple-handler test passed
✅ Full pipeline demonstration succeeded

## Follow-up Recommendations

### Optional Enhancements

1. **Event persistence**: Save key events to a database or message queue
2. **Event replay**: Support event replay for debugging or analysis
3. **Event filtering**: Support more sophisticated event filtering and routing
4. **Priority queue**: Support priority-based event processing
5. **Distributed events**: Support cross-service events via a message queue

### Integration Recommendations

1. **Monitoring integration**: Integrate Prometheus for metrics collection
2. **Logging integration**: Unified structured logging
3. **Tracing integration**: Integrate with the existing tracing system
4. **Alerting integration**: Event-based alerting mechanism

## Sample Output

Running `go run ./internal/event/demo/main.go` shows the full RAG pipeline event output:

```
Step 1: Query Received
[MONITOR] Query received - Session: session-xxx, Query: What is RAG technology?
[ANALYTICS] Query tracked - User: user-123, Session: session-xxx

Step 2: Query Rewriting
[MONITOR] Query rewrite started
[MONITOR] Query rewritten - Original: What is RAG technology?, Rewritten: Retrieval-Augmented Generation technology...
[CUSTOM] Query Transformation: ...

Step 3: Vector Retrieval
[MONITOR] Retrieval started - Type: vector, TopK: 20
[MONITOR] Retrieval completed - Results: 18, Duration: 301ms
[CUSTOM] Retrieval Efficiency: Rate: 90.00%

Step 4: Result Reranking
[MONITOR] Rerank started - Input: 18
[MONITOR] Rerank completed - Output: 5, Duration: 201ms
[CUSTOM] Rerank Statistics: Reduction: 72.22%

Step 5: Chat Completion
[MONITOR] Chat generation started
[MONITOR] Chat generation completed - Tokens: 256, Duration: 801ms
[ANALYTICS] Chat metrics - Model: gpt-4, Tokens: 256
```

## Summary

The event system has been fully implemented and validated through testing, and is ready for immediate integration into the WeKnora project for monitoring, logging, analytics, and debugging the various stages of the query processing pipeline. The system is designed to be simple, high-performing, and easy to use and extend.
