package database

import (
	"context"
	"database/sql"
	"embed"

	_ "github.com/mattn/go-sqlite3"
	"github.com/ztrue/tracerr"

	"github.com/HomewireApp/homewire/internal/logger"
	"github.com/go-gorp/gorp"
	migrate "github.com/rubenv/sql-migrate"
)

//go:embed migrations
var fsMigrations embed.FS

type Database struct {
	ctx        context.Context
	connString string
	db         *sql.DB
	dbmap      *gorp.DbMap
}

func Open(path string) (*Database, error) {
	db, err := sql.Open("sqlite3", path)

	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	dbmap := &gorp.DbMap{Db: db, Dialect: gorp.SqliteDialect{}}

	dbmap.AddTableWithName(IdentityModel{}, "identities")
	dbmap.AddTableWithName(WireModel{}, "wires")

	return &Database{
		db:         db,
		dbmap:      dbmap,
		connString: "sqlite3://" + path,
		ctx:        context.Background(),
	}, nil
}

func (db *Database) Migrate() error {
	m := migrate.EmbedFileSystemMigrationSource{
		FileSystem: fsMigrations,
		Root:       "migrations",
	}

	applied, err := migrate.Exec(db.db, "sqlite3", m, migrate.Up)
	if err != nil {
		return err
	}

	logger.Debug("Successfully executed %v migrations", applied)

	return nil
}

func (db *Database) GetConnection() (*sql.Conn, error) {
	return db.db.Conn(db.ctx)
}

func (db *Database) Find(holder interface{}, query string, args ...interface{}) ([]interface{}, error) {
	return db.dbmap.Select(holder, query, args...)
}

func (db *Database) Insert(model interface{}) error {
	return db.dbmap.Insert(model)
}

func (db *Database) Close() error {
	return db.db.Close()
}
