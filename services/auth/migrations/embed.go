// Package migrations — SQL-миграции auth-service (goose, embedded).
//
// Миграции компилируются в бинарник: сервис сам приводит свою схему БД
// к актуальному виду при старте.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
