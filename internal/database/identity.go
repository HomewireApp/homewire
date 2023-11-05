package database

import (
	"time"

	"github.com/HomewireApp/homewire/internal/utils"
	"github.com/go-gorp/gorp"
	"github.com/ztrue/tracerr"
)

type IdentityModel struct {
	Id            string    `db:"id"`
	CreatedAt     time.Time `db:"created_at"`
	Name          string    `db:"name"`
	PrivateKey    []byte    `db:"private_key"`
	DisplayedName string    `db:"displayed_name"`
}

func (i *IdentityModel) PreInsert(s gorp.SqlExecutor) error {
	i.CreatedAt = time.Now().UTC()
	return nil
}

func (db *Database) GetOrCreateIdentity(name string) (*IdentityModel, error) {
	var results []IdentityModel

	if _, err := db.Find(&results, "SELECT * FROM identities WHERE name = ?", name); err != nil {
		return nil, tracerr.Wrap(err)
	}

	prvBytes, err := utils.GeneratePrivateKeyBytes()
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	var result *IdentityModel

	if len(results) > 0 {
		result = &results[0]
	} else {
		result = &IdentityModel{
			Id:            utils.NewUUID(),
			Name:          name,
			PrivateKey:    prvBytes,
			DisplayedName: utils.NewPetName(),
		}

		err := db.Insert(result)
		if err != nil {
			return nil, tracerr.Wrap(err)
		}
	}

	return result, nil
}
