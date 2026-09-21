# HTTP Request Lifecycle & `ServeHTTP` Flow

In Go's `net/http` package, everything revolves around a single universal interface:
```go
type Handler interface {
    ServeHTTP(ResponseWriter, *Request)
}
```
Every component in the request pipeline—the server engine, middleware, router (`ServeMux`), and endpoint handlers—implements `ServeHTTP`. The request moves from one component to another through a chain of `ServeHTTP` calls.

---

## Section 1: Conceptual View (High-Level Mental Model)

This is how the flow works conceptually using common web architecture terms:

```
[ Incoming HTTP Request ]
          │
          ▼
   1. Go HTTP Server
          │
          ▼ calls ServeHTTP()
   2. Middleware (e.g., RequestID, Logging, Auth)
          │
          ▼ calls next.ServeHTTP() (where next = mux)
   3. ServeMux (Router)
          │
          ▼ matches URL pattern & calls ServeHTTP()
   4. URL Handler (e.g., GetListings, Healthz)
```

### Step-by-Step Breakdown:

1. **Request arrives at the Go Server:** The server accepts the incoming network connection.
2. **Server calls `handler.ServeHTTP`:** The server delegates the request to whatever root handler was registered (either `mux` directly, or a global middleware wrapping `mux`).
3. **Middleware executes & passes forward:** 
   * When setting up the application in `main.go`, we create `mux` first and pass it into the middleware initialization (`middleware.RequestIDMiddleware(mux)`).
   * When a request arrives, the middleware executes its own logic first (e.g., generating a request ID, validating tokens, or injecting values into `r.Context()`).
   * Then it calls `next.ServeHTTP(w, r.WithContext(ctx))`. Because `next` holds our `mux`, this hands execution directly over to the router.
4. **`ServeMux` matches the route:** `mux.ServeHTTP` inspects the request's URL path and HTTP method (e.g., `"GET /listings"`), searches its routing table, and finds the corresponding registered handler.
5. **`ServeMux` calls `urlHandler.ServeHTTP`:** The router invokes the endpoint handler, which runs your actual business logic, interacts with the database, and writes the HTTP response back to `w`.

---

## Section 2: Deep Dive Under the Hood (Go Standard Library Internals)

This section shows the exact code path and function calls inside Go's `net/http/server.go`.

### 1. `srv.ListenAndServe()`
When you start the server in your `main.go`:
```go
srv := http.Server{
    Addr:    ":" + cfg.Port,
    Handler: middleware.RequestIDMiddleware(mux),
}
srv.ListenAndServe()
```
Inside `net/http/server.go`:
* It opens the TCP socket: `ln, err := net.Listen("tcp", addr)`
* It hands the listener over to `srv.Serve(ln)`

---

### 2. `srv.Serve(l net.Listener)`
The server runs an infinite accept loop:
```go
for {
    rw, err := l.Accept() // Accepts incoming TCP connection
    c := srv.newConn(rw)
    go c.serve(connCtx)   // Spawns a dedicated goroutine for this connection!
}
```

---

### 3. `c.serve(ctx context.Context)`
Inside the connection's goroutine:
* It parses the raw HTTP request bytes from the wire into `http.Request`.
* It then calls:
```go
serverHandler{c.server}.ServeHTTP(w, w.req)
```

---

### 4. `serverHandler.ServeHTTP(rw, req)` (~Line 2910 in `server.go`)
This is the bridge where Go's network engine enters your application code:
```go
func (sh serverHandler) ServeHTTP(rw ResponseWriter, req *Request) {
    handler := sh.srv.Handler // Reads srv.Handler configured in main.go
    if handler == nil {
        handler = DefaultServeMux
    }
    // ...
    handler.ServeHTTP(rw, req) // 🔥 Calls your root handler!
}
```
If you wrapped `mux` in `RequestIDMiddleware`, `handler` is your middleware.

---

### 5. `RequestIDMiddleware.ServeHTTP(w, r)`
Inside your middleware:
```go
func RequestIDMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // 1. Middleware work
        rid := r.Header.Get("X-Request-ID")
        if rid == "" {
            rid = generateRequestID()
        }
        ctx := context.WithValue(r.Context(), requestIDKey, rid)

        // 2. Pass to next (which is mux!)
        next.ServeHTTP(w, r.WithContext(ctx)) // 🔥 Calls mux.ServeHTTP!
    })
}
```

---

### 6. `mux.ServeHTTP(w, r)` (~Line 2680 in `server.go`)
Inside `ServeMux`:
```go
func (mux *ServeMux) ServeHTTP(w ResponseWriter, r *Request) {
    // 1. Matches the request to your registered route patterns
    h, _ := mux.Handler(r)

    // 2. Calls ServeHTTP on the matched handler!
    h.ServeHTTP(w, r) // 🔥 Calls your handler's ServeHTTP!
}
```

---

### 7. `http.HandlerFunc.ServeHTTP(w, r)` (~Line 2362 in `server.go`)
Your handlers (like `listingHandler.GetListings` or `handlers.Healthz`) were registered with `mux.HandleFunc(...)`. Go converted them to `http.HandlerFunc`:
```go
type HandlerFunc func(ResponseWriter, *Request)

func (f HandlerFunc) ServeHTTP(w ResponseWriter, r *Request) {
    f(w, r) // 🔥 Executes your actual function!
}
```
Finally, your endpoint function executes, queries the database, encodes JSON, and sends the response back down the TCP connection.

---

## Summary Call Stack Cheat Sheet

```
1. net.Listener.Accept()
2. go c.serve()
3. serverHandler.ServeHTTP(w, req)
4. srv.Handler.ServeHTTP(w, req)          --> Your Middleware
5. next.ServeHTTP(w, r.WithContext(ctx))  --> mux.ServeHTTP()
6. mux.Handler(r).ServeHTTP(w, r)         --> http.HandlerFunc.ServeHTTP()
7. listingHandler.GetListings(w, r)       --> Your Handler Logic
```
