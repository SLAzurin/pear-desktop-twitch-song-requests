package databaseconn

import (
	"database/sql"
	"log"
	"os"

	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/data"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

const dbName = "pear-desktop-twitch-song-requests.db"

func init() {
	// create db if missing
	if _, err := os.Stat(dbName); os.IsNotExist(err) {
		// file does not exist
		file, err := os.Create(dbName)
		if err != nil {
			log.Fatal(err.Error())
		}
		file.Close()
	}
}

func NewDBConnection() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "file:"+dbName)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func getMigrator() (*migrate.Migrate, error) {
	var m *migrate.Migrate
	var err error
	d, err := iofs.New(data.GetMigrationFS(), "iofs/migrations")
	if err != nil {
		return nil, err
	}
	m, err = migrate.NewWithSourceInstance("iofs", d, "sqlite3://"+dbName)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func Migrate() error {
	// create tables if not exist
	var m *migrate.Migrate
	var err error
	m, err = getMigrator()
	if err != nil {
		return err
	}
	uperr := m.Up()
	if uperr != nil && uperr != migrate.ErrNoChange {
		m.Close()
		// attempt recovery force down
		log.Println("Migration failed, recovering...")

		dB, err := NewDBConnection()
		if err != nil {
			log.Println("Migration recovery failed to connect to database")
			return err
		}
		stmt, err := dB.Prepare("SELECT version, dirty FROM schema_migrations LIMIT 1")
		if err != nil {
			log.Println("Migration recovery failed to prepare statement to get database migration version")
			dB.Close()
			return err
		}
		version := uint64(0)
		dirty := false

		row := stmt.QueryRow()
		err = row.Scan(&version, &dirty)
		if err != nil {
			log.Println("Migration recovery failed to fetch db version and dirty state")
			dB.Close()
			return err
		}
		if !dirty {
			log.Println("Migration recovery not necessary, schema is not dirty")
			dB.Close()
			return uperr
		}
		dB.Close()

		// close db to allow migrate engine to take over
		m, err = getMigrator()
		if err != nil {
			return err
		}
		defer m.Close()
		forceVersion := version - 1
		log.Println("Migration recovery will force schema back 1 version, to version", forceVersion)
		err = m.Force(int(forceVersion))
		if err != nil {
			log.Println("Migration recovery failed to force version to", forceVersion)
			return err
		}
		log.Println("Migration recovery forced version to", forceVersion)
		return uperr
	}
	m.Close()
	return nil
}
