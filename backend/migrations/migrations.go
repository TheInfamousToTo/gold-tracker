// Package migrations carries the schema SQL into the binary.
//
// The files sit here rather than at the repository root because the
// backend Dockerfile copies only the backend directory, and go:embed
// cannot reach outside its own package. Embedding them means the
// deployed image can never be running one schema while the repository
// describes another.
package migrations

import "embed"

//go:embed *.sql
var Files embed.FS
