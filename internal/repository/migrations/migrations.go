package migrations

import "embed"

//go:embed sqlite/*.sql
var EmbedFS embed.FS
