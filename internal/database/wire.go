package database

import (
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ztrue/tracerr"
)

type WireModel struct {
	Id         string    `db:"id"`
	CreatedAt  time.Time `db:"created_at"`
	Name       string    `db:"name"`
	PrivateKey []byte    `db:"private_key"`
	OtpSecret  []byte    `db:"otp_secret"`
}

func (w *WireModel) PreInsert(s gorp.SqlExecutor) error {
	w.CreatedAt = time.Now().UTC()
	return nil
}

func (db *Database) FindAllWires() ([]*WireModel, error) {
	var knownWires []*WireModel
	_, err := db.dbmap.Select(&knownWires, "SELECT * FROM wires")
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	return knownWires, nil
}
