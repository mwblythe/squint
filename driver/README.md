# Squint SQL Driver

## Overview

Because of the way variadic functions work, you cannot literally pass the return values of `Build()` directly into `Exec()` or `Query()`.

```go
// NOPE: you can't do this
err := db.Exec(b.Build("insert into users", newUser))

// instead, you have to do this
sql, binds := b.Build("insert into users", newUser)
err := db.Exec(sql, binds...)
```

To make this more convenient, you can use the **squint** `driver`. This will let you call the standard `sql` functions directly, using squint `Build()` syntax.

## Installation

The `driver` package is included in the **squint** module.

```
go get github.com/mwblythe/squint
```

## Usage

```go
package main

import (
  "database/sql"
  "log"
  _ "modernc.org/sqlite"
  "github.com/mwblythe/squint/driver"
)

func init() {
  // register a squint-enabled sqlite driver
  driver.Register("sqlite")
}

func main() {
  // open the database via the squint driver
  db, err := sql.Open("squint-sqlite", "file::memory:")
  if err != nil {
    log.Fatal(err)
  }
  defer db.Close()

  // execute a query in the style of squint Builder
  rows, err := db.Query(
    "select id, name from users where id in", []int{10, 20, 30},
 	  "and active = 1",
  )
}
```

## Limitations

The standard `sql` query functions require a `string` before the binds. This means you can't pass things like a `Builder` option or condition first. Instead, start with a fragment of your query, then anything compatible with `Builder` can come after:

```go
// this won't compile
db.Query(squint.OmitEmpty(), "insert into users", newUser)

// but this will
db.Query("insert into users", squint.OmitEmpty(), newUser)
```

## Options

`Regsiter()` accepts options that let you customize behavior. Default values shown in parenthesis.

| Option              | Purpose                           | Default                  |
| ------------------- | --------------------------------- | ------------------------ |
| `Name(string)`      | Name to use for the squint driver | `"squint-" + toDriver`   |
| `Builder(*Builder)` | squint `Builder()` to use         | result of `NewBuilder()` |

For example:

```go
driver.Register("sqlite",
  driver.Name("squintlite"),
  driver.Builder(
    squint.NewBuilder(squint.Log(true)),
  ),
)
```

## Connection Settings

Because the `squint` driver acts as a proxy, connection settings must be in sync on both sides. So, please use the following functions instead of the `sql.DB` methods.

```go
SetConnMaxIdleTime(*sql.DB, time.Duration)
SetConnMaxLifetime(*sql.DB, time.Duration)
```

This will set the corresponding value on both your `sql.DB` handle as well as the underlying one that is wrapped by the `squint` proxy driver.

## See Also

The separate [squint-driver-tests](https://github.com/mwblythe/squint-driver-tests) module has compatibility tests for various databases.
