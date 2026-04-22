# AI Agent Guidelines: Conevent Backend

## 0 — Overview, Purpose & Language Policy

This document provides strict guidelines for AI coding agents working on the **Conevent** backend. These rules ensure
maintainability, safety, performance, and developer velocity in Go.

**Language Policy (CRITICAL):**

* **Source Code:** All code (variables, functions, types, logs, and code comments) **MUST** be written in **English**.
* **Documentation:** The `README.md` and high-level project documentation **MUST** be written in **Portuguese (PT-BR)**.

**Rule strictness:**

* **MUST**: Enforced by CI/review; no exceptions.
* **SHOULD**: Strong recommendations; deviate only with documented rationale.
* **CAN**: Allowed patterns without extra approval.

---

## 1 — Project Context

- **Project Name:** Conevent (Backend)
- **Language:** Go (Golang)
- **Web Framework:** [Fiber](https://gofiber.io/) (High performance, Express-like).
- **Database:** PostgreSQL with [pgx](https://github.com/jackc/pgx) driver.
- **SQL Layer:** [sqlc](https://sqlc.dev/) (Type-safe code generation from SQL).
- **Goal:** Provide a robust, concurrent, and secure backend foundation for the Conevent application.

## 2 — Project Structure

We follow the standard Go project layout:

```text
/ (root)
├── cmd/
│   └── conevent/        # Main application entry point (main.go)
├── internal/            # Private application and business logic
│   ├── api/             # Fiber handlers & routes
│   ├── db/              # sqlc generated code & migrations
│   │   ├── queries/     # .sql files for sqlc
│   │   └── schema/      # Database schema definitions
│   └── service/         # Business logic / Use cases
├── pkg/                 # Public library code (safe to import by other projects)
├── config/              # Configuration loading and structures
├── sqlc.yaml            # sqlc configuration
├── go.mod / go.sum      # Dependency management
└── README.md            # General project documentation (PT-BR)
```

## 3 — Development Commands

```bash
# Run the application locally
go run ./cmd/conevent

# Run tests with race detector
go test -race ./...

# Generate Go code from SQL queries using sqlc
sqlc generate

# Run linters (ensure golangci-lint is installed)
golangci-lint run

# Update dependencies
go mod tidy
```

---

## 4 — Core Go Rules (Agent Directives)

### Architecture & Dependencies

- **ARC-1 (MUST)** Ask clarifying questions if the API shape or data flow is ambiguous before generating code.
- **ARC-2 (SHOULD)** Keep the `main.go` file minimal. It should only load configs, wire dependencies (Dependency
  Injection), and start the server.
- **ARC-3 (SHOULD)** Prefer the standard library (`net/http`, `encoding/json`, `crypto`) over external dependencies
  unless there is a significant payoff.

### Code Style (`gofmt` standard)

- **CS-1 (MUST)** Avoid stutter in naming. E.g., Use `user.Service`, not `user.UserService`. Use `config.Load()`, not
  `config.LoadConfig()`.
- **CS-2 (MUST)** Use input structs for functions receiving more than 3 arguments to ensure extensibility.
- **CS-3 (SHOULD)** Define small interfaces close to the consumer (where they are used), not where they are implemented.
- **CS-4 (SHOULD)** Keep exported (public) API surfaces as small as possible. Return concrete types, but accept
  interfaces.

### Error Handling

- **ERR-1 (MUST)** Never ignore errors. Handle them or return them.
- **ERR-2 (MUST)** Add context when bubbling up errors using `%w`: `fmt.Errorf("fetching event %s: %w", id, err)`.
- **ERR-3 (MUST)** Use `errors.Is` or `errors.As` for checking error types. Never use string matching (
  `strings.Contains(err.Error(), "...")`).
- **ERR-4 (SHOULD)** Define domain-specific sentinel errors in their respective packages (e.g.,
  `ErrEventNotFound = errors.New("event not found")`).

### Concurrency & Goroutines

- **CC-1 (MUST)** Tie every goroutine's lifetime to a `context.Context` to prevent memory leaks.
- **CC-2 (MUST)** Protect all shared mutable state with `sync.Mutex` or `sync/atomic`. No "probably safe" races.
- **CC-3 (MUST)** The **sender** is responsible for closing channels; receivers must never close them.
- **CC-4 (SHOULD)** Use `golang.org/x/sync/errgroup` for fan-out work to easily catch errors and cancel sibling
  goroutines.

### Contexts

- **CTX-1 (MUST)** If a function requires a context, `ctx context.Context` must be the **first** parameter.
- **CTX-2 (MUST)** Never store `context.Context` inside a struct type. Pass it explicitly to methods.
- **CTX-3 (MUST)** Honor context cancellations (`ctx.Done()`) inside long-running loops or blocking operations.

### SOLID Principles (Go Idiomatic)

- **SLD-1 (MUST) Single Responsibility (SRP):** Packages, structs, and functions must have one single reason to change.
  Strict separation of concerns is required: Handlers deal ONLY with HTTP/Fiber, Services deal ONLY with business logic,
  and DB/SQLC deals ONLY with data persistence.
- **SLD-2 (SHOULD) Open/Closed (OCP):** Extend behavior through struct composition and interface satisfaction, rather
  than modifying existing complex functions with endless `if/else` or `switch` flags.
- **SLD-3 (MUST) Liskov Substitution (LSP):** When implementing an interface, the concrete type must fulfill the
  interface's implicit contract without causing unexpected panics or breaking the consumer's logic.
- **SLD-4 (MUST) Interface Segregation (ISP):** Prefer small, focused interfaces (1 to 3 methods) over large, monolithic
  ones. Interfaces must be defined **close to the consumer** (where they are used), not where they are implemented.
- **SLD-5 (MUST - CRITICAL) Dependency Inversion (DIP):** High-level business logic (Services) MUST NOT depend on
  low-level implementations (database layers, external APIs). They must depend on abstractions (Interfaces). Always use
  Dependency Injection (e.g., passing a repository interface into a service constructor).

---

## 5 — APIs & HTTP Handlers

- **API-1 (MUST)** Set explicit timeouts on the Fiber app config (`ReadTimeout`, `WriteTimeout`, `IdleTimeout`) to
  prevent slow-loris attacks.
- **API-2 (MUST)** Use standard structured JSON responses for both success and error payloads.
- **API-3 (SHOULD)** Decouple HTTP handlers from business logic. Handlers should only parse requests, call a
  Service/UseCase, and format the response.

### 5.1 — Fiber Framework Rules

- **FBR-1 (MUST)** Do not use standard `net/http` patterns. Use Fiber's `*fiber.Ctx` for all request/response handling.
- **FBR-2 (MUST - CRITICAL)** The `*fiber.Ctx` is **NOT thread-safe** and its values are pooled/recycled as soon as the
  handler returns. **Never** pass `*fiber.Ctx` to a new goroutine. If background processing is needed, extract the
  necessary values (strings, bytes) beforehand or use `c.Copy()`.
- **FBR-3 (SHOULD)** Use Fiber's built-in parsing and response methods (e.g., `c.BodyParser(&struct)`, `c.JSON(data)`,
  `c.Status()`) instead of standard `encoding/json` inside the handlers.
- **FBR-4 (SHOULD)** Fiber handlers should remain thin. Extract parameters, call the Service/UseCase layer, and return
  the response. Do not put business logic inside the Fiber handler.
- **FBR-5 (CAN)** Utilize Fiber's native middleware (e.g., `recover`, `logger`, `cors`, `limiter`) via
  `github.com/gofiber/fiber/v2/middleware/...` instead of building custom ones.

---

## 6 — Database & Storage Rules

- **DB-1 (MUST)** Use `sqlc` with the `pgx` driver for PostgreSQL. Do not use ORMs like `gorm` or generic mappers like
  `sqlx`. All database interactions must happen through `sqlc` generated code.
- **DB-2 (MUST - CRITICAL)** Write pure SQL in the `internal/db/queries/` directory. Use parameterized queries (`$1`,
  `$2`) to prevent SQL injection. Never concatenate strings to build SQL queries.
- **DB-3 (MUST)** Always pass `context.Context` to database calls (e.g., `queries.WithTx(tx).GetEvent(ctx, id)`) to
  ensure queries are cancelled if the HTTP request is aborted by the client.
- **DB-4 (MUST)** Configure the database connection pool explicitly (`SetMaxOpenConns`, `SetMaxIdleConns`,
  `SetConnMaxLifetime`).
- **DB-5 (SHOULD)** Keep database transactions as short as possible. Do not make external HTTP calls or heavy
  computations while holding an open transaction.

---

## 7 — Configuration, Secrets & Environment Variables

- **ENV-1 (MUST)** Centralize all configuration in a strongly-typed `Config` struct (e.g., within the `config/`
  package). Do not scatter `os.Getenv()` calls throughout the codebase.
- **ENV-2 (MUST - CRITICAL)** Fail fast. The application MUST crash on startup (`panic` or `log.Fatal`) if a required
  environment variable (e.g., `DATABASE_URL`, `JWT_SECRET`) is missing or invalid. Do not start the server in a broken
  state.
- **ENV-3 (MUST - CRITICAL)** Never hardcode secrets or commit `.env` files to version control. Always provide a
  `.env.example` file with dummy values for developer onboarding.
- **ENV-4 (SHOULD)** Use lightweight libraries like `github.com/joho/godotenv` to load `.env` files locally, but rely on
  native environment injection (pure `os.Getenv`) in production environments.
- **ENV-5 (MUST)** Pass the `Config` struct explicitly via Dependency Injection to the handlers, services, and
  repositories that need it. Never use global variables to store the configuration.
- **SEC-1 (MUST)** Never log sensitive data (PII, passwords, tokens, API keys, or database connection strings).
- **SEC-2 (MUST)** Validate and sanitize all incoming user input at the boundaries (handlers) before passing it to the
  business logic.

---

## 8 — Testing & Observability

- **TST-1 (MUST)** Use Table-Driven Tests for testing multiple scenarios (inputs/outputs) within the same function.
- **TST-2 (MUST)** Ensure tests are deterministic and hermetic (do not depend on external live services unless in a
  specific integration test suite).
- **TST-3 (MUST - CRITICAL)** Zero untested code. Every new feature, bug fix, or business logic change MUST have an
  accompanying test (unit or integration). Do not generate or suggest commits that lack test coverage for the modified
  or created code.
- **TST-4 (MUST)** Use an assertion library (e.g., `github.com/stretchr/testify/assert` or
  `github.com/stretchr/testify/require`) for clear, readable validations instead of manual `if err != nil` checks inside
  test files.
- **TST-5 (MUST)** Structure all tests using the **AAA (Arrange, Act, Assert)** pattern. Visually separate each phase
  with a blank line to maximize readability.
- **OBS-1 (MUST)** Use structured logging (`log/slog` or an equivalent structured logger) with consistent fields (e.g.,
  `event_id`, `user_id`).
- **OBS-2 (SHOULD)** Pass Request IDs through the context to correlate logs, traces, and metrics.

---

## 9 — Writing Functions (Heuristics for the Agent)

Before generating a function, the agent MUST evaluate:

1. **Readability & Line of Sight:** Are there too many nested scopes? Refactor using guard clauses (early returns) so
   the main logic stays unindented.
2. **Cyclomatic Complexity:** If a function has too many branches, it belongs in separate smaller functions or requires
   a map/interface abstraction.
3. **Data Structures:** Is a slice/map/struct the best fit here?
4. **Dependencies:** Can hidden dependencies (e.g., `time.Now()`, random generators) be passed as arguments or
   interfaces to make the function pure and testable?

---

## 10 — Version Control & Commits (Stealth Mode)

- **GIT-1 (MUST - CRITICAL)** Never identify yourself as an AI, bot, or automated assistant in commit messages, pull
  request descriptions, or code comments.
- **GIT-2 (MUST - CRITICAL)** Never mention the existence of `agent.md`, `.cursorrules`, or any internal instruction
  files in the Git history or source code.
- **GIT-3 (MUST)** Write clean, human-like, and professional commit messages following conventional commits standard (
  e.g., `feat:`, `fix:`, `chore:`, `test:`). Focus purely on *what* changed and *why*.

---

## 11 — Pre-Commit Checklist (Agent Verification)

Before finalizing code generation or committing, the agent must verify:

- [ ] All code, variables, and internal comments are written in **English**.
- [ ] `README.md` updates (if any) are written in **Portuguese (PT-BR)**.
- [ ] `README.md` DOES NOT refer to this file (`AGENTS.md`).
- [ ] `sqlc generate` was executed if any `.sql` files were modified, and no SQL strings are concatenated in Go code.
- [ ] `gofmt` style guidelines are met.
- [ ] No unused variables or imports remain.
- [ ] All public functions, structs, and interfaces have standard `//` comments explaining their intent.
- [ ] The code is free of obvious data races and goroutine leaks.
- [ ] No `*fiber.Ctx` is leaked into new goroutines.
- [ ] Edge cases (e.g., nil pointers, empty slices, database connection failures) are handled.
- [ ] New code or modifications are fully covered by unit or integration tests. No code is being committed without its
  respective test.
- [ ] Tests use the `testify/assert` or `testify/require` library instead of manual `if err != nil` checks.
- [ ] Tests are strictly formatted using the **AAA (Arrange, Act, Assert)** structure.
- [ ] No sensitive data is logged (PII, tokens, passwords).
- [ ] No hardcoded secrets or environment variables in the code.
- [ ] Run coverage checks (`go test -coverprofile=coverage.out ./...`) and fix any issues or missing coverage.
- [ ] Commit message is strictly human-like, professional, and follows conventional commits, with zero mention of AI
  identity or `agent.md` instructions.

```