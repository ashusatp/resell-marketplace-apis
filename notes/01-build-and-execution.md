# Go Build & Execution Notes

## 1. `go run` vs `go build`

| Command | What It Does | Best For |
| :--- | :--- | :--- |
| `go run main.go` | Compiles code into a **temporary directory** in the background, executes it, and removes the temp files. | **Local Development & quick testing** |
| `go build -o bin/main main.go` | Compiles the code into a **permanent, standalone binary** executable (e.g., in `bin/main`). | **Production & Deployments** |

---

## 2. Production Workflow

In production, you never run source code with `go run`. Instead:
1. **Build the binary:**
   ```bash
   go build -o bin/main main.go
   ```
2. **Execute the binary:**
   ```bash
   ./bin/main
   ```

### Why this is essential for Production:
- **Zero Dependencies:** The production server or container doesn't need Go installed—just the compiled binary.
- **Fast Startup:** Instant startup time with no compilation overhead.
- **Minimal Footprint:** Allows building ultra-lean Docker images (e.g., using multi-stage builds with `FROM scratch` or `FROM alpine`).

---

## 3. Cross-Compilation (`GOOS` & `GOARCH`)

Go compiles directly to native machine code for the target OS and CPU architecture. Go makes cross-compiling for other platforms simple using environment variables:

```bash
# Build for Linux (64-bit amd64) - Standard cloud/server deployment
GOOS=linux GOARCH=amd64 go build -o bin/main-linux main.go

# Build for Windows (64-bit)
GOOS=windows GOARCH=amd64 go build -o bin/main.exe main.go

# Build for macOS (Apple Silicon ARM64)
GOOS=darwin GOARCH=arm64 go build -o bin/main-mac main.go
```
