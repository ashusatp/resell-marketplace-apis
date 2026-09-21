# Use of Context in Go

In Go, the `context` package (`context.Context`) is one of the most fundamental abstractions for concurrent and network programming. It carries deadlines, cancellation signals, and request-scoped values across API boundaries and between goroutines.

In an HTTP service, every incoming `*http.Request` comes with its own request-scoped context (`r.Context()`).

---

## 1. Cancellation Signal

### What it does
A cancellation signal informs all downstream operations (database queries, external HTTP calls, worker goroutines) that their work is no longer needed and they should stop immediately.

### Why use it
* **Avoid Zombie Queries:** If an HTTP client (browser, mobile app, frontend) disconnects, closes the tab, or aborts the request before the server finishes responding, the HTTP server cancels `r.Context()`.
* **Free System Resources:** When you pass `ctx` to database operations (such as `DB.QueryContext` or `DB.ExecContext`), the database driver listens to `ctx.Done()`. Upon cancellation, it aborts the in-flight SQL query on PostgreSQL and returns the database connection back to the connection pool immediately instead of letting it run uselessly.

### Example in HTTP Handlers:
```go
func (lh *ListingHandler) DeleteListing(w http.ResponseWriter, r *http.Request) {
    // 1. Get request-scoped context
    ctx := r.Context()

    // 2. Pass ctx to DB operation. If client aborts, the query is canceled immediately!
    result, err := lh.DB.ExecContext(ctx, `DELETE FROM listings WHERE id = $1`, id)
    if err != nil {
        // err will indicate context cancellation if the client disconnected
        lh.Logger.Error("DeleteListing exec error", "err", err)
        http.Error(w, "internal server error", http.StatusInternalServerError)
        return
    }
    // ...
}
```

### Manual Cancellation Pattern:
```go
ctx, cancel := context.WithCancel(parentCtx)
defer cancel() // Ensures resources are freed when done
```

---

## 2. Deadline & Timeout

### What it does
Sets a hard upper limit on how long an operation or chain of operations is allowed to take.

* **`context.WithTimeout(parent, duration)`**: Cancels automatically after a given duration (e.g., 3 seconds).
* **`context.WithDeadline(parent, time)`**: Cancels automatically when a specific point in time is reached.

### Why use it
* **Fail-Fast SLAs:** Prevents hanging queries, deadlocks, or slow external third-party services from blocking goroutines and exhausting connection pools indefinitely.
* **Predictable Latency:** Ensures the server returns an error (or fallback) within a guaranteed timeframe instead of leaving callers waiting forever.

### Example:
```go
// Create a context with a 2-second timeout derived from the request context
ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
defer cancel() // Crucial: always call cancel() to avoid leaking context resources

rows, err := lh.DB.QueryContext(ctx, query)
if err != nil {
    if errors.Is(ctx.Err(), context.DeadlineExceeded) {
        http.Error(w, "query timed out", http.StatusGatewayTimeout)
        return
    }
    http.Error(w, "database error", http.StatusInternalServerError)
    return
}
```

---

## 3. Request-Bound Values

### What it does
Allows passing request-scoped data down through middleware, handlers, and services without having to modify function signatures to pass extra arguments.

### Common Use Cases
* **Authentication / Authorization:** Attaching authenticated user details (e.g., `userID`, `role`) extracted by auth middleware.
* **Observability & Tracing:** Storing a unique `request_id` or distributed trace span ID so logs can be correlated across services.
* **Tenant Isolation:** Multi-tenant identifiers (`tenantID`).

### Best Practices for `context.WithValue`
1. **Never use plain strings or built-in types as keys:** Always define custom, unexported types for context keys to avoid collisions with other packages.
2. **Only for request-scoped data:** Do not use context values to pass optional function parameters, business logic configuration, or database handles. Context values should be strictly metadata tied to the lifetime of the request.

### Example:
```go
// Define custom unexported key type
type contextKey string

const userIDKey contextKey = "userID"

// 1. In Middleware: inject value into context
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        userID := "user-123" // Extracted from JWT / session token

        ctx := context.WithValue(r.Context(), userIDKey, userID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// 2. In Handler: retrieve value from context
func (lh *ListingHandler) CreateListing(w http.ResponseWriter, r *http.Request) {
    userID, ok := r.Context().Value(userIDKey).(string)
    if !ok {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }
    // Proceed with userID...
}
```

---

## Summary Matrix

| Feature | Primary Function | Typical Go Function |
| :--- | :--- | :--- |
| **1. Cancellation Signal** | Stop work when client disconnects or parent aborts | `context.WithCancel`, `r.Context()` |
| **2. Deadline / Timeout** | Limit maximum execution time to avoid hangs | `context.WithTimeout`, `context.WithDeadline` |
| **3. Request-Bound Values**| Carry request-scoped metadata (auth, trace IDs) | `context.WithValue` |
