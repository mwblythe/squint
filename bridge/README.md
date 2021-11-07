# Squint Bridge

‼️ The newer [driver](../driver) package is preferred. However, if `bridge` better fits your needs, press on.

## Overview

Because of the way variadic functions work, you cannot literally pass the return values of `Build()` directly into `Exec()` or `Query()`.

```go
// NOPE: you can't do this
err := db.Exec(b.Build("insert into users", newUser))

// instead, you have to do this
sql, binds := b.Build("insert into users", newUser)
err := db.Exec(sql, binds...)
```

To make this more convenient, you can use a Squint bridged database.

## Installation

The `bridge` package is included in the **squint** module.

```
go get github.com/mwblythe/squint
```

## Usage

```go
import (
  _ "modernc.org/sqlite"  
  "github.com/jmoiron/sqlx"
  "github.com/mwblythe/squint"
  "github.com/mwblythe/squint/bridge"  
)

// open database and bridge it
con, err := sqlx.Open("sqlite", "file::memory:")
db := bridge.NewDB(con, squint.NewBuilder())

// now queries are easier via the Squint extension
err := db.Squint.Exec("insert into user", newUser)

// but you can still do things the old way too
err := db.Exec("update users set balance = ? where id = ?", 0.00, 10)
```

The bridge has wrapper functions for the standard `Exec`, `Query` and `QueryRow`, as well as the `sqlx` extensions `MustExec`, `Queryx`, `QueryRowx`, `Get` and `Select`. Wherever those original functions expect `sql` and `binds`, the `Squint` versions expect Squint `Build()` parameters.

The bridged database also returns a bridged transaction if you call `Begin`, `Beginx`, or `MustBegin`:

```go
// using the above bridged database
tx := db.Begin()

// use the same Squint extensions
if err := tx.Squint.Exec("insert into user", newUser); err == nil {
  db.Commit()
} else {
  db.Rollback()
}
```

## See Also

For a more flexible and seamless solution, see the squint [driver](../driver) package.