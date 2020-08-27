# Squint - An interpolating SQL builder

> Inspired by Perl's `SQL::Interp`

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

Generally, pointers are dereferenced and their values used as if they were passed directly. If the pointer is `nil`, it will map to a `NULL` value. Pointers can be useful in a `struct` as discussed below with the `KeepNil` option.

### Conditions

When crafting a complex query, you sometimes need to build it up in bits while checking various conditions. Was an ID specified? Was extra information requested? While you can do this by carefully filling an array that you then pass to `Build()`, Squint has another option.

```go
b.Build(
  "SELECT u.* FROM users u",
  b.If(boolCondition, "JOIN employees e ON u.id = e.id"),
  "WHERE u.id IN", ids,
)
```

You can include any number of arguments in `If()`, and they will only be processed by `Build()` if the condition is true.

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

The `Builder` has a few options to control behavior. They and their defaults are:

```go
Tag       string // tag name for field mapping ("db")
KeepNil   bool   // keep nil struct/map field values? (false)
KeepEmpty bool   // keep empty string struct/map field values? (false)
```

These can all be set directly on the `Builder`:

```
b := squint.NewBuilder()
b.KeepNil = true
```

A bit more about `KeepNil` and `KeepEmpty`:

When a struct or map is processed, any string values that are empty (`""`) will be skipped if `KeepEmpty` is false. Any values that are `nil` will be skipped if `KeepNil` is false. This is the default behavior. Why?

It's common to have a `struct` type that represents a full set of possible columns to use.  It's also common that only some of those values are supplied in a given scenario. For example:

```go
// the columns we allow to be updated
type Updates struct {
  FirstName  string
  LastName   string
  Department string
}

// update a user record
func updateUser(id int, updates *Updates) error {
  b := squint.NewBuilder()
  sql, binds := b.Build("update users set", updates, "where id =", id)
  _, err := db.Exec(sql, binds...)
  return err
}
```

What if only `updates.Department` is set? The zero value for a `string` is the empty string `("")`. If `KeepEmpty` is `true`, the user record would be updated with an empty `FirstName` and `LastName`. This is generally not what you want. If `KeepEmpty` is `false`, then only `Department` will be included in the update. This safer behavior is the default.

Note that `KeepEmpty` only applies to `strings`. This is because the zero values of other basic types are more common as real values, zero (`0`) in particular. So you can't really tell if one of these was explicltly set or not. How to handle this?

If you're sure the zero value is not a value you want saved, you can set `omitempty` in a particular field's `db` tag.

```go
type Updates struct {
  Name string
  Age  int    `db:"omitempty"`
}
```

Otherwise, you can leverage the `KeepNil` option. It is essentially the same as `KeepEmpty`, but applies to pointers. So, you can do:

```go
type Updates struct {
  FirstName  string
  LastName   string
  Department string
  Balance    *float64
}
```

Now you can tell the difference between setting `Balance` to zero or not setting it at all. With `KeepNil` set to `false`, an empty `Balance` would be skipped; otherwise it would be included as a SQL `NULL`.

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
