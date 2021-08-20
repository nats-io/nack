package schemas

import (
	"embed"
)

//go:embed jetstream
//go:embed server

var schemas embed.FS

func Load(schema string) ([]byte, error) {
	return schemas.ReadFile(schema)
}
