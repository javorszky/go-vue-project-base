# Backend guidelines

## Technology choices

- **HTTP framework**: [labstack/echo](https://echo.labstack.com/)
- **Logging**: `log/slog` (standard library)
- **Observability**: OpenTelemetry — traces, metrics, and logs (via the Go OTel SDK)

## Common commands

```bash
# Build
go build ./...

# Test
go test ./cmd/... ./internal/...

# Run a single test
go test ./path/to/package -run TestFunctionName

# Lint (if golangci-lint is installed)
golangci-lint run
```

## Security headers

The backend serves only JSON API responses — it does not serve `index.html` or frontend assets, so it does not set a page-level `Content-Security-Policy`. That is the static host's responsibility (see the frontend guidelines).

Every API response must include these headers, applied globally via an Echo middleware:

| Header | Value |
|--------|-------|
| `X-Content-Type-Options` | `nosniff` |
| `X-Frame-Options` | `DENY` |
| `Referrer-Policy` | `strict-origin-when-cross-origin` |
| `Cache-Control` | `no-store` (for authenticated/sensitive endpoints) |

CORS headers (`Access-Control-Allow-Origin`, etc.) are also set here — allow only the known frontend origin, never `*` in production.

## Coding style and engineering choices

Follow [Effective Go](https://go.dev/doc/effective_go) throughout. Key rules:

### Formatting
- Run `gofmt` (or `goimports`) on all code. No exceptions.
- Opening brace on the same line as the statement — never on its own line.

### Naming
- Packages: lowercase, single word, no underscores.
- Multi-word identifiers: `MixedCaps` or `mixedCaps` — never underscores.
- Getter methods: no `Get` prefix — `obj.Owner()` not `obj.GetOwner()`. Setter prefix is `Set`.
- Single-method interfaces: use `-er` suffix (`Reader`, `Writer`, `Formatter`).
- Avoid stutter: `ring.New` not `ring.NewRing`, `bufio.Reader` not `bufio.BufReader`.

### Language usage
- Use the latest available Go syntax (project targets Go 1.26+).
- Prefer slices over arrays; use `make` for slices, maps, and channels.
- Use composite literals with named fields; return pointers to local structs freely.
- Design zero values to be useful and ready to use without further initialization.
- Omit `else` when the `if` branch ends in `return`, `break`, `continue`, or `goto`.
- Use `range` for iteration; discard unused loop variables with `_`.
- Named return values are acceptable when they meaningfully document the function signature.
- Use `defer` for all cleanup (close, unlock, etc.) — guarantees execution on every return path.

### Errors
- Return `error` as the last return value; never panic for recoverable conditions.
- Error strings are lowercase and don't end with punctuation (`"open file: not found"`).
- Return structured error types (not bare strings) when callers need to inspect details.
- Use `panic` only for truly unrecoverable states (programmer errors at init time).

### Interfaces
- Keep interfaces small — one or two methods is idiomatic. Shallow interfaces are easier to mock in tests; a five-method interface usually signals that a dependency should be split.
- Define interfaces in the consuming package, not the implementing package.
- Constructors should return interface types, not concrete types, when the interface is the intended API surface.
- Use compile-time interface checks where helpful: `var _ io.Reader = (*MyType)(nil)`.

### Concurrency
- Share memory by communicating (channels), not by communicating by sharing memory.
- Prefer channels over mutexes for coordinating goroutines.
- Size channel buffers deliberately; unbuffered channels are the safe default.

### Context
- Pass `context.Context` as the **first parameter** to every function that does I/O, calls external services, or may block: `func Do(ctx context.Context, ...) error`.
- Never store a context in a struct; always pass it explicitly through the call chain.
- Attach deadlines and timeouts at the outermost entry point (e.g. HTTP handler), not deep in the stack: `ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second); defer cancel()`.
- Always call the `cancel` function returned by `WithCancel`, `WithTimeout`, and `WithDeadline` — use `defer cancel()` immediately after creation to avoid context leaks.
- Respect cancellation in all blocking operations: select on `ctx.Done()` alongside channel sends/receives, and check `ctx.Err()` in loops before continuing work.
- Propagate `ctx.Err()` (or a wrapped form of it) upward when a context is cancelled or timed out — don't swallow it.
- Use `context.WithoutCancel` (Go 1.21+) when you need to do cleanup work that must outlive a cancelled context (e.g. flushing spans to the OTel exporter).

### Graceful shutdown
- Trap OS signals using `signal.NotifyContext` (Go 1.16+) — listen for `syscall.SIGINT`, `syscall.SIGTERM`, and `syscall.SIGHUP` at minimum. Note: `SIGKILL` cannot be caught; design for the signals that can be.
  ```go
  ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
  defer stop()
  ```
- When the signal arrives, begin an orderly shutdown sequence with a bounded timeout so the process never hangs indefinitely:
  ```go
  shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
  defer cancel()
  ```
- Shutdown order matters — tear down in reverse initialisation order:
  1. Stop accepting new requests (`echo.Shutdown(shutdownCtx)`).
  2. Wait for in-flight HTTP handlers to finish (Echo's `Shutdown` does this).
  3. Flush and shut down the OTel trace/metric/log exporters.
  4. Close database connections and any other resources.
- Use `context.WithoutCancel` for the shutdown context so cancellation of the root context (triggered by the signal) does not abort the shutdown work itself.
- Log the reason for shutdown and each major step at `slog.Info` level; log errors at `slog.Error`.
- If the shutdown timeout is exceeded, log a warning and allow the process to exit — do not block indefinitely.
- The main goroutine should block on the signal context, then drive the shutdown sequence directly rather than relying on `init`-time hooks scattered across packages.

### Imports
- Side-effect-only imports use the blank identifier: `import _ "net/http/pprof"`.
- Group imports: stdlib, then external, then internal — separated by blank lines.

### Testing
- All test files use the external test package: `package foo_test`, not `package foo`. This is black-box testing — tests interact with the package only through its exported API, exactly as a real caller would.
- Unexported functions must be covered indirectly: write tests against the exported functions and methods that exercise them. If an unexported function cannot be reached through any exported path, that is a design smell — consider whether it should be exported, inlined, or removed.
- Never reach into unexported identifiers by putting tests in the same package just to gain access. If a test genuinely cannot be written without internal access, refactor the design first.
- Use table-driven tests: define a `[]struct{ name string; ... }` slice of cases and range over it with `t.Run(tc.name, func(t *testing.T) { ... })`. This keeps assertions uniform and makes it trivial to add new cases.
- Use [`github.com/stretchr/testify`](https://github.com/stretchr/testify) for assertions (`assert`, `require`) and mocks (`mock`). Use `require` when a failure makes the rest of the test meaningless (e.g. nil check before dereferencing); use `assert` otherwise to let all assertions in a case run.
- Generate mocks with `testify/mock` via `mockery`. Mock only at package boundaries (interfaces you own or consume from external packages) — do not mock concrete types.
- Never call `os.Setenv` / `os.Unsetenv` in tests to configure the app under test. Use `config.LoadFrom(map[string]string{...})` instead — it parses from an in-memory map and requires no cleanup.
