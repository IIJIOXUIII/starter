// Package migrations — SQL-миграции users-service (goose, embedded).
//
// Нумерация с 10000: goose_db_version общая с auth-service (одна БД в dev),
// версии auth начинаются с 1.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
