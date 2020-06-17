package squint

// TODO: should this be a separate module so that the base
// squint doesn't have a dependency on sqlx?

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
)

// Target is the destination of a Bridge.
// It is typically a database or transaction handle.
type Target interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row

	MustExec(query string, args ...interface{}) sql.Result
	Queryx(query string, args ...interface{}) (*sqlx.Rows, error)
	QueryRowx(query string, args ...interface{}) *sqlx.Row
	Get(dest interface{}, query string, args ...interface{}) error
	Select(dest interface{}, query string, args ...interface{}) error
}

// Bridge provides a connection between a Builder and a Target.
// It essentially wraps a Builder interface around standard query functions.
//
// So something like this:
//
// sql, binds := builder.Build("SELECT * FROM table WHERE", conditions)
// rows, err  := db.Query(sql, binds)
//
// Becomes:
//
// rows, err := bridge.Query("SELECT * FROM table WHERE", conditions)
//
type Bridge struct {
	*Builder
	target Target
}

func (b *Bridge) Exec(bits ...interface{}) (sql.Result, error) {
	sql, binds := b.Build(bits...)
	return b.target.Exec(sql, binds...)
}

func (b *Bridge) Query(bits ...interface{}) (*sql.Rows, error) {
	sql, binds := b.Build(bits...)
	return b.target.Query(sql, binds...)
}

func (b *Bridge) QueryRow(bits ...interface{}) *sql.Row {
	sql, binds := b.Build(bits...)
	return b.target.QueryRow(sql, binds...)
}

func (b *Bridge) MustExec(bits ...interface{}) sql.Result {
	sql, binds := b.Build(bits...)
	return b.target.MustExec(sql, binds...)
}

func (b *Bridge) Queryx(bits ...interface{}) (*sqlx.Rows, error) {
	sql, binds := b.Build(bits...)
	return b.target.Queryx(sql, binds...)
}

func (b *Bridge) QueryRowx(bits ...interface{}) *sqlx.Row {
	sql, binds := b.Build(bits...)
	return b.target.QueryRowx(sql, binds...)
}

func (b *Bridge) Get(dest interface{}, bits ...interface{}) error {
	sql, binds := b.Build(bits...)
	return b.target.Get(dest, sql, binds...)
}

func (b *Bridge) Select(dest interface{}, bits ...interface{}) error {
	sql, binds := b.Build(bits...)
	return b.target.Select(dest, sql, binds...)
}

// DB is a bridged db connection; Create one with squint.BridgeDB
type DB struct {
	*sqlx.DB
	Squint Bridge
}

// Tx is a bridged transaction; Create one with DB.Begin(x)
type Tx struct {
	*sqlx.Tx
	Squint Bridge
}

// BridgeDB creates a bridge between a database handle and a Builder
func BridgeDB(db *sqlx.DB, builder *Builder) *DB {
	return &DB{db, Bridge{builder, db}}
}

// Begin starts and bridges a transaction
func (db *DB) Begin() (*Tx, error) {
	return db.Beginx()
}

// bridgeTX creates a new bridge for the supplied Tx
func (db *DB) bridgeTx(tx *sqlx.Tx) *Tx {
	return &Tx{tx, Bridge{db.Squint.Builder, tx}}
}

// Beginx starts and bridges a transaction
func (db *DB) Beginx() (*Tx, error) {
	if tx, err := db.DB.Beginx(); err == nil {
		return db.bridgeTx(tx), nil
	} else {
		return nil, err
	}
}

// Beginx starts and bridges a transaction, but panics on error
func (db *DB) MustBegin() *Tx {
	return db.bridgeTx(db.DB.MustBegin())
}
