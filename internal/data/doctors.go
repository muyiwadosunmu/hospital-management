package data

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"log/slog"
	"time"
)

type Doctor struct {
	ID        int64     `json:"id"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	Email     string    `json:"email"`
	Password  password  `json:"-"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"-"`
}

type DoctorModel struct {
	DB *sql.DB
}

func (s *DoctorModel) Create(ctx context.Context, user *Doctor) error {
	query := `INSERT INTO doctors (first_name,last_name, email, password) 
	VALUES ($1, $2, $3, $4) RETURNING id, first_name, last_name, created_at`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	err := s.DB.QueryRowContext(ctx, query, user.FirstName, user.LastName, user.Email, user.Password.hash).
		Scan(&user.ID, &user.FirstName, &user.LastName, &user.CreatedAt)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return ErrDuplicateEmail
		case err.Error() == `pq: duplicate key value violates unique constraint "users_username_key"`:
			return ErrDuplicateUsername
		default:
			return err
		}
	}

	return nil
}

func (s *DoctorModel) GetById(ctx context.Context, id int64) (*Doctor, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}
	query := `SELECT id, first_name, last_name, email, created_at, updated_at 
    FROM doctors 
    WHERE id = $1`
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	user := &Doctor{}
	err := s.DB.QueryRowContext(ctx, query, id).
		Scan(&user.ID,
			&user.FirstName,
			&user.LastName,
			&user.Email,
			&user.CreatedAt,
			&user.UpdatedAt)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	slog.Info("doctor retrieved", "user", user)
	return user, nil
}

func (s *DoctorModel) delete(ctx context.Context, tx *sql.Tx, id int64) error {
	query := `DELETE FROM users WHERE id = $1`
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	_, err := tx.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	return nil
}

func (s *DoctorModel) getUserFromInvitation(ctx context.Context, tx *sql.Tx, token string) (*Receptionist, error) {
	query := `SELECT u.id, u.first_name, u.last_name, u.email, u.created_at, 
	FROM users u 
	JOIN user_invitations ui ON u.id = ui.user_id 
	WHERE ui.token = $1 AND ui.expiry > $2
	`
	hash := sha256.Sum256([]byte(token))
	hashedToken := hex.EncodeToString(hash[:])

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	user := &Receptionist{}
	err := tx.QueryRowContext(ctx, query, hashedToken, time.Now()).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Email, &user.CreatedAt)

	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return user, nil

}

// func (s *ReceptionistModel) update(ctx context.Context, tx *sql.Tx, user *Receptionist) error {
// 	query := `UPDATE users SET first_name = $1, email = $2, is_active = $3  WHERE id = $4`

// 	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
// 	defer cancel()

// 	_, err := tx.ExecContext(ctx, query, user.FirstName, user.Email, user.IsActive, user.ID)
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

func (s *DoctorModel) deleteUserInvitation(ctx context.Context, tx *sql.Tx, _ int64) error {
	return nil
}

func (s *DoctorModel) GetDocByEmail(ctx context.Context, email string) (*Doctor, error) {
	query := `
		SELECT id, first_name, email, password, created_at FROM doctors
		WHERE email = $1
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	user := &Doctor{}
	err := s.DB.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.FirstName,
		&user.Email,
		&user.Password.hash,
		&user.CreatedAt,
	)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return user, nil
}
