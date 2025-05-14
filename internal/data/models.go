package data

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var (
	ErrRecordNotFound    = errors.New("record not found")
	ErrEditConflict      = errors.New("edit conflict")
	QueryTimeoutDuration = time.Second * 5
)

type Models struct {
	Receptionists ReceptionistModel
	Doctors       DoctorModel
	Patients      PatientModel
	Roles         RoleModel
}

func NewModels(db *sql.DB) Models {
	return Models{
		Receptionists: ReceptionistModel{db},
		Doctors:       DoctorModel{db},
		Patients:      PatientModel{db},
		Roles:         RoleModel{db},
	}
}

func withTx(db *sql.DB, ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}
