package migrations

import "embed"

// Files contains all SQL migrations embedded at build time.
//
//go:embed *.sql
var Files embed.FS
