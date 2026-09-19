# Custom `ServeMux` vs `http.DefaultServeMux` (Security & Best Practices)

## 1. The Question
In `cmd/api/main.go`, why do we create our own router instance with:
```go
mux := http.NewServeMux()
```
instead of using the global helper methods directly like:
```go
http.HandleFunc("GET /healthz", handler)
http.ListenAndServe(":8080", nil) // nil = use default router
```

---

## 2. The Core Problem: `http.DefaultServeMux` is a Global Variable

When you don't create your own `ServeMux`, Go routes all requests through `http.DefaultServeMux`.

Because it is an exported **global variable** in the `net/http` package, **any package in your entire dependency tree has access to it**.

```
                           +------------------------+
                           | http.DefaultServeMux   | (Global, public to all packages)
                           +------------------------+
                             ^                    ^
                             |                    |
                 Your App Routes                Third-Party Packages
             (e.g., /healthz, /products)    (can inject hidden endpoints!)
```

---

## 3. The Security Risk (Supply Chain & Unintended Endpoints)

### A. Third-Party Dependency Route Injection
If you import any third-party library, its `init()` function runs automatically before your `main()`. A malicious, rogue, or poorly-designed package could register routes on `http.DefaultServeMux` without your knowledge:

```go
// Hidden inside some imported third-party package:
func init() {
    http.HandleFunc("/admin/backdoor", exploitHandler)
}
```
If you use `http.DefaultServeMux`, that backdoor endpoint is **live and exposed on your public server**!

### B. The Classic `net/http/pprof` Danger
A famous real-world example in Go is the standard debugging tool `net/http/pprof`.
Simply importing it:
```go
import _ "net/http/pprof"
```
automatically registers internal profiling endpoints (`/debug/pprof/*`) to `http.DefaultServeMux`. If `DefaultServeMux` is exposed to the internet, attackers can view memory dumps, stack traces, and internal environment configurations.

---

## 4. The Solution: Scoped `http.NewServeMux()`

By instantiating an isolated router:

```go
// 1. Create a scoped, private mux instance
mux := http.NewServeMux()

// 2. Explicitly register ONLY your intended routes
mux.HandleFunc("GET /healthz", healthCheckHandler)

// 3. Attach this isolated mux to your HTTP server
srv := &http.Server{
    Addr:    ":" + cfg.Port,
    Handler: mux, // <- Only this mux handles traffic
}
```

### Key Advantages:
1. **Security Isolation:** External packages cannot alter, intercept, or register endpoints on your private `mux` instance.
2. **No Global State Pollution:** Avoids shared mutable global state across packages.
3. **Modular & Testable:** You can easily create isolated `ServeMux` instances for unit tests or run multiple servers on different ports with completely different route sets (e.g., public API on port `8080`, internal metrics on port `9090`).
