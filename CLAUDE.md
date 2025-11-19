# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Squint is an interpolating SQL builder for Go that allows SQL and bind variables to be intermixed in their natural order. It's NOT an ORM - it's a query building assistant that works with the standard `database/sql` package.

The project has two main components:
1. **Core Builder** (`squint.go`, `query.go`, `options.go`) - The heart of Squint that processes SQL fragments and variables into queries and binds
2. **Driver Package** (`driver/`) - A proxy SQL driver that enables using Builder syntax directly in standard `database/sql` query functions

## Development Commands

### Testing
```bash
# Run all tests in the main package
go test -v

# Run tests in the driver package
cd driver && go test -v

# Run a specific test
go test -v -run TestSquint

# Run tests with coverage
go test -cover
```

### Linting
The project uses golangci-lint with configuration in `.golangci.toml`:
```bash
golangci-lint run
```

### Building
Standard Go build commands apply:
```bash
go build
```

## Architecture

### Core Builder Flow

The `Builder` accepts variadic arguments and processes them based on type and context:

1. **query.go** contains the state machine logic:
   - `sqlState` tracks context (base, insert, set, in clauses)
   - Regular expressions (`insertRX`, `setRX`, `inRX`) detect SQL context
   - Different types are handled specially based on current state

2. **Type Processing**:
   - Strings are treated as SQL fragments by default
   - Basic types (int, bool, etc.) become bind placeholders
   - Strings can be forced to binds via `&myString` or `squint.Bind(myString)`
   - Arrays/slices flatten inline OR expand for IN clauses (context-dependent)
   - Structs/maps expand to column=value pairs (WHERE, SET, INSERT contexts)
   - Pointers are dereferenced, nil becomes NULL

3. **State-Aware Processing**:
   - INSERT context: structs/maps become `(col1, col2) VALUES (?, ?)`
   - SET context: structs/maps become `col1=?, col2=?`
   - IN context: arrays expand to `(?, ?, ?)`
   - WHERE context (default): structs/maps become `col1=? AND col2=?`

### Driver Architecture

The driver package (`driver/`) implements a proxy pattern:

1. **sqDriver** wraps another SQL driver (e.g., "mysql", "sqlite")
2. **sqConn** wraps `*sql.Conn` and intercepts query methods
3. Queries are processed through Builder before passing to underlying driver
4. Connection pooling is maintained via `sync.Map` keyed by DSN

Key limitation: Standard `sql` functions require first argument to be a string, so Builder options/conditions must come after the initial SQL fragment.

### Empty Value Handling

Complex logic in `query.go` for handling zero values in structs/maps:

- **KeepEmpty** (default): Include zero values
- **OmitEmpty**: Skip zero values
- **NullEmpty**: Convert zero values to SQL NULL
- Custom handlers via `EmptyFn`
- Per-field overrides via struct tags: `db:"omitempty"`, `db:"keepempty"`, `db:"nullempty"`
- Special case: Multi-row inserts force KeepEmpty for consistent column counts

### Field Mapping

Struct fields map to database columns:
- Default: Use field name verbatim
- Tag override: `db:"column_name"`
- Skip field: `db:"-"`
- Empty handling: `db:"column_name,omitempty"`

Mapping logic is in the `sift()` function in `query.go`.

## Testing Patterns

Tests use `github.com/stretchr/testify` for assertions. The driver tests use `github.com/DATA-DOG/go-sqlmock` for mocking database interactions.

Test files:
- `squint_test.go` - Core Builder functionality
- `compat_test.go` - Backward compatibility for deprecated options
- `driver/driver_test.go` - Driver wrapper functionality

## Module Information

- Module: `github.com/mwblythe/squint`
- Go version: 1.14+
- Dependencies: testify, go-sqlmock (test only)

## Important Patterns

### Bind Placeholder Functions

Four built-in styles via `BindFn`:
- `bindQuestion()`: `?` (MySQL, SQLite)
- `bindDollar()`: `$1, $2` (PostgreSQL, SQLite)
- `bindAt()`: `@p1, @p2` (SQL Server)
- `bindColon()`: `:b1, :b2` (Oracle)

Custom bind functions can be registered via `WithBindFn()`.

### Conditional Query Building

The `Condition` type (created via `If()`) allows inline conditionals:
```go
b.Build(
  "SELECT * FROM users",
  b.If(includeEmployees, "JOIN employees e ON u.id = e.id"),
  "WHERE id IN", ids,
)
```

### Driver Registration

The `Register()` function in `driver/driver.go` creates a new SQL driver by wrapping an existing one. The wrapped driver name follows the pattern `"squint-" + originalDriver` by default.

Connection settings must be synchronized between outer and inner connections via special helpers: `SetConnMaxIdleTime()` and `SetConnMaxLifetime()`.
