// filepath: internal/db/migrations/embed.go
package migrations

import "embed"

// FS embeds all SQL migration files in this directory.
//
//go:embed *.sql
var FS embed.FS
