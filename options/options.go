package options

import (
	"os"
	"path"

	"github.com/HomewireApp/homewire/logger"
	"github.com/ztrue/tracerr"
)

type Options struct {
	DatabasePath    string
	ProfileName     string
	MigrateDatabase bool
	Logger          logger.Logger
}

func Default() (*Options, error) {
	userHome, err := os.UserHomeDir()

	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	hwHome := path.Join(userHome, ".homewire")

	if err := os.MkdirAll(hwHome, os.ModePerm); err != nil {
		return nil, tracerr.Wrap(err)
	}

	dbPath := path.Join(hwHome, ".homewire.db")

	return &Options{
		DatabasePath:    dbPath,
		ProfileName:     "default",
		MigrateDatabase: true,
		Logger:          logger.DefaultConsoleConfig().CreateLogger("homewire"),
	}, nil
}

func (o *Options) WithDatabasePath(path string) *Options {
	o.DatabasePath = path
	return o
}

func (o *Options) WithProfileName(profileName string) *Options {
	o.ProfileName = profileName
	return o
}

func (o *Options) WithMigrateDatabase(migrateDatabase bool) *Options {
	o.MigrateDatabase = migrateDatabase
	return o
}

func (o *Options) WithLogger(logger logger.Logger) *Options {
	o.Logger = logger
	return o
}
