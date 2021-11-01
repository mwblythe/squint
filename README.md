# Squint - An interpolating SQL builder

## Overview

The `database/sql` package is very capable but somewhat tedious to use. You must hand-write full SQL queries with bind placeholders and provide a matching ordered list of variables. It's familiar, but inconvenient and repetitive. Squint makes things easier by allowing SQL and bind variables to be intermixed in their natural order. It also interpolates the variables into the proper bind placeholders and values, including complex types like structs and maps.  Squint is not an ORM, though. It's merely a pleasant query building assistant.

## Builder

The `Builder` is the heart of Squint. It accepts a list of SQL fragments and variables and returns the resulting SQL statement and bind variables.

```go
import "github.com/mwblythe/squint"

// a simple example

idList := []int{10, 20, 30}
b := squint.NewBuilder()
sql, binds := b.Build("select * from users where id in", idList, "and active = 1")
rows, err := db.Query(sql, binds...)
```

The `sql` and `binds` returned are ready to pass into the standard `database/sql` query functions, such as `Exec`, `Query` and `QueryRow`. (See [Bridge](#bridge) for an easier way)

### Basic Variables

A `string` by default is treated as SQL. Other variables of the [basic types](https://github.com/golang/example/tree/master/gotypes#basic-types) will transform into a SQL bind placeholder (`?`) and an accompanying bind value. To have a `string` treated as a bind variable, you can pass a reference (`&myString`) or cast it as `squint.Bind(myString)`.

```go
// treated as binds
b.Build("select * from users where id =", 10) // good
b.Build("select * from users where is_active =", true) // good

// strings are sql without special handling
name := "Frank"
b.Build("select * from users where name =", name) // bad
b.Build("select * from users where name =", &name) // good
b.Build("select * from users where name =", squint.Bind(name)) // good
```

### Arrays and Slices

By default, arrays and slices will be flattened and treated as if their contents had been passed inline. However, as part of a SQL  `IN` clause, they will be expanded into appropriate placeholders and binds (even `strings`).

```go
bits := []string{"select *", "from", "crew"}
b.Build(bits, "where id =", 10) // slice is flattened

names := []string{"jim", "spock", "uhura"}
b.Build("select * from crew where name in", names) // magic
```

*Note that only arrays of basic types are supported for `IN` clause.*

### Structs and Maps

By default, these will be expanded in the style of a `WHERE` clause (`column = ?`) and joined with `AND`.

```go
// easily build where clauses from structs
type Query struct {
  Origin     string
  Completed  bool
}

q := Query{Origin: "online", Completed: false}
b.Build("select * from orders where", q)

// or maps
type M map[string]interface{}
q := M{"origin": "online", "completed": false }
b.Build("select * from orders where", q)
```

There is special handling for `INSERT` statements:

```go
// simple user struct
type User struct {
  Id   int
  Name string
}

// fill structure, likely from user input
newUser := User{10, "Frank"}

// build our query
b.Build("insert into users", newUser)

// multi-row insert
users := []User{
  { 10, "Frank"},
  { 20, "Hank"},
}

b.Build("insert into users", users)

// also handles maps
type Item map[string]interface{}
b.Build("insert into users", Item{"id": 10, "name": "Frank"})
```

Also for `UPDATE` statements:

```go
// use a structure
type Updates struct {
  Role       string
  Department string
}

updates := Updates{"captain", "housewares"}
b.Build("UPDATE user SET", updates, "WHERE id =", id)

// or a map
type Updates map[string]interface{}
updates := Updates{"role": "captain", "department": "housewares"}
b.Build("UPDATE user SET", updates, "WHERE id =", id)
```

### Pointers

Generally, pointers are dereferenced and their values used as if they were passed directly. If the pointer is `nil`, it will map to a `NULL` value. Pointers can be useful in a `struct` as discussed below under "Empty Values".

### Conditions

When crafting a complex query, you sometimes need to build it up in bits while checking various conditions. Was an ID specified? Was extra information requested? While you can do this by carefully filling an array that you then pass to `Build()`, Squint has another option.

```go
b.Build(
  "SELECT u.* FROM users u",
  b.If(boolCondition, "JOIN employees e ON u.id = e.id"),
  "WHERE u.id IN", ids,
)
```

You can include any number of arguments in `If()`, and they will only be processed by `Build()` if the condition is true. This can also be called as `squint.If()`

### Field Mapping

When mapping `struct` fields into database columns, by default the names are used verbatim.  You can change the mapping by using the `db` struct field.

```go
type User struct {
  Id        int                            // column is Id
  FirstName string `db:"first_name"`       // column is first_name
  Username  string `db:"-"`                // skip this one
  ManagerId int    `db:"mgr_id,omitempty"` // skip if empty (0)
}
```

*A custom mapping function may be a future enhancement*

### Options

The `Builder` uses functional options to control behavior. They and their defaults are:

```go
Tag(string)       // tag name for field mapping ("db")
KeepEmpty()       // keep empty values in struct/map
OmitEmpty()       // omit empty values in struct/map
NullEmpty()       // treat empty values as nulls in struct/map
LogQuery(bool)    // log queries (false)
LogBinds(bool)    // log bind values from queries (false)
Log(bool)         // shorthand to log both queries AND binds (false)
```

These can all be set via `NewBuilder()`:

```
b := squint.NewBuilder(
  squint.NullEmpty(),
  squint.Log(true),
)
```

They can also be set via `Build()`, and will only be in effect for that query:

```go
b.Build(
  squint.LogBinds(false),
  "update users set password =", &myPass,
  "where id =", id,
)
```

### Empty Values

When a struct or map is processed, empty (Go "zero") values need special consideration. You can control how they are treated on a builder level with the `KeepEmpty()`, `OmitEmpty()`, and `NullEmpty()` options. These are mutually exclusive, so only the last one used will win. Each of these options has a struct field equivalent for selective override:

```go
type Person struct {
  Name string
  Age  int        `db:"omitempty"`
  NumChildren int `db:"keepempty"`
  ManagerID int   `db:"nullempty"`
}
```

Since the empty value for pointers is `nil`, you can sometimes leverage this:

```go
type Updates struct {
  Department string
  Balance    *float64 `db:"omitempty"`
}
```

Now you can tell the difference between setting `Balance` to `0` or not setting it at all. With `omitempty`,  an empty `Balance` would be skipped, but a pointer to a `0` would be kept.

**NOTE:** If `OmitEmpty()` is in effect for a multi-row inserts, `KeepEmpty()` will be used instead. This is because the column count must be consistent across rows. `NullEmpty()` and `KeepEmpty()` will be used as set.

## Bridge

Because of the way variadic functions work, you cannot literally pass the return values of `Build()` directly into something like `Exec()`.

```go
// NOPE: you can't do this
err := db.Exec(b.Build("insert into user", newUser))

// instead, you have to do this
sql, binds := b.Build("insert into user", newUser)
err := db.Exec(sql, binds...)
```

To make this more convenient, you can use a Squint database `Bridge`.

```go
// open database and build a bridge
con, err := sqlx.Open("mysql", dsn)
db := squint.BridgeDB(con, squint.NewBuilder())

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
