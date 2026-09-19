# Makefile as a Task Runner in Go

## 1. What is a `Makefile`?
A `Makefile` acts as a task runner/automation tool. Instead of repeatedly typing long commands like:
```bash
go build -o bin/main main.go
./bin/main
```
You define short target commands inside a file named `Makefile`, allowing you to just run:
```bash
make build
make run
```

---

## 2. The Golden Rule of Makefiles ⚠️
> **IMPORTANT:** In Makefiles, command lines **MUST be indented with a real TAB (`\t`)**, **NOT SPACES**. If you use spaces, `make` will throw a syntax error (`*** missing separator`).

---

## 3. Standard Go `Makefile` Structure

```makefile
# Build the application binary
build:
	go build -o bin/main main.go

# Run the compiled binary
run: build
	./bin/main

# Clean up built artifacts
clean:
	rm -rf bin
```

---

## 4. How It Works
- `make build`: Runs the `go build -o bin/main main.go` command and outputs the binary in `bin/main`.
- `make run`: First runs `build` (because `build` is listed as a dependency after `run:`), then executes `./bin/main`.
- `make clean`: Removes the `bin` directory so you can start fresh.
