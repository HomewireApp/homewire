package options

import (
	"os"
	"path"

	"github.com/ztrue/tracerr"
)

type Options struct {
	DatabasePath    string
	ProfileName     string
	MigrateDatabase bool
}

func Default() (*Options, error) {
	home, err := os.UserHomeDir()

	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	dbPath := path.Join(home, ".homewire.db")

	return &Options{
		DatabasePath:    dbPath,
		ProfileName:     "default",
		MigrateDatabase: true,
	}, nil
}
