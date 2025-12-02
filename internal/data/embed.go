package data

import (
	"embed"
)

//go:embed iofs/migrations/*
var migrationsFS embed.FS

func GetMigrationFS() *embed.FS {
	return &migrationsFS
}
