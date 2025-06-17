package sqlite

import (
	"embed"
	"io/fs"
	"path"
)

const migrationsDir = "migrations"

var (
	//go:embed migrations/*.sql
	migrations                            embed.FS
	schemaVersion, schemaVersionReadError = GetSchemaVersion()
)

func getSchemaFiles() ([]fs.DirEntry, error) {
	return migrations.ReadDir(migrationsDir)
}

func GetSchemaVersion() (int, error) {
	files, err := getSchemaFiles()
	if err != nil {
		return -1, err
	}

	return len(files), nil
}

func GetMigrations() ([]string, error) {
	var statements []string

	files, err := getSchemaFiles()
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		fileName := path.Join(migrationsDir, file.Name())
		sql, err := migrations.ReadFile(fileName)
		if err != nil {
			return nil, err
		}

		statements = append(statements, string(sql))
	}

	return statements, nil
}
