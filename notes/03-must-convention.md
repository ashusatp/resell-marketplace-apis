# The `Must` Convention in Go (`MustLoad`, `MustCompile`, etc.)

## 1. What is the `Must` Prefix?

In Go, idiomatic functions return `(Value, error)`. However, when you see a function prefixed with **`Must`** (e.g., `MustLoad()`, `MustCompile()`, `MustParse()`):

> **Rule of Thumb:** A `Must...` function **does not return an error**. Instead, it **panics** if something goes wrong.

```go
// Standard Go pattern (returns error):
func Load() (Config, error)

// "Must" convention (panics on failure, returns only the value):
func MustLoad() Config
```

---

## 2. Why Use `MustLoad()` for Configuration?

### The "Fail-Fast" Principle (Crash Early, Crash Loudly)
Configuration loading happens at application startup in `main()`. If essential environment variables like `PORT`, `ENV`, or `DATABASE_URL` are missing:
- The server **cannot function properly**.
- Continuing execution would lead to unexpected `nil` pointer exceptions, broken connections, or silent failures later in production.
- Calling `panic()` immediately stops the server right at boot-up with a clear, readable stack trace pinpointing the exact missing config.

### Cleaner Code in `main()`
Using `MustLoad()` eliminates repetitive `if err != nil` boilerplates in `main.go`:

```go
func main() {
    // Clean and direct: If this fails, the process stops immediately
    cfg := config.MustLoad()

    // Server setup continues safely knowing cfg is guaranteed valid
    ...
}
```

---

## 3. Go Standard Library Examples

The `Must` naming convention is a well-established pattern across Go's standard library:

| Standard Function | Purpose | Behavior |
| :--- | :--- | :--- |
| `regexp.MustCompile(pattern)` | Compiles a regular expression | Panics if the regex pattern string is invalid syntax |
| `template.Must(tmpl, err)` | Parses an HTML/text template | Panics if template parsing fails |

---

## 4. Golden Rules: When to Use vs. Avoid `Must`

| When TO Use `Must...` ✅ | When NOT to Use `Must...` ❌ |
| :--- | :--- |
| During **startup / initialization** (in `main.go`, `init()`, or config loading). | Inside **HTTP request handlers**, background jobs, or controllers. |
| For resources the app **cannot live without** (env variables, DB connection string, essential templates). | For operations handling **user input** or dynamic runtime data. |
| When crashing immediately is safer than running in an invalid state. | When an error can be handled, logged, or returned as an HTTP response (e.g., `400 Bad Request` or `500 Internal Server Error`). |
