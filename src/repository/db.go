package repository

import (
	"database/sql"
)

// DBTX abstracts both *sql.DB and *sql.Tx, allowing the same query methods
// to be used in both normal operation and within transactions (e.g., tests).
type DBTX interface {
	QueryRow(query string, args ...interface{}) *sql.Row
	Query(query string, args ...interface{}) (*sql.Rows, error)
	Exec(query string, args ...interface{}) (sql.Result, error)
}
