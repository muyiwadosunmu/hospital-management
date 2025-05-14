package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/muyiwadosunmu/hospital-management/internal/validator"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrDuplicateEmail    = errors.New("a user with that email already exists")
	ErrDuplicateUsername = errors.New("a user with that username already exists")
)

type password struct {
	plaintext *string
	hash      []byte
}

type Receptionist struct {
	ID        int64     `json:"id"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	Email     string    `json:"email"`
	Password  password  `json:"-"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"-"`
}

type ReceptionistModel struct {
	DB *sql.DB
}

func (p *password) Set(plaintextPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintextPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	p.plaintext = &plaintextPassword
	p.hash = hash
	return nil
}

func (p *password) Compare(text string) error {
	return bcrypt.CompareHashAndPassword(p.hash, []byte(text))
}

func (s *ReceptionistModel) Create(ctx context.Context, user *Receptionist) error {
	query := `INSERT INTO receptionists (first_name,last_name, email, password) 
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

func (s *ReceptionistModel) GetById(ctx context.Context, id int64) (*Receptionist, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}
	query := `SELECT id, first_name, last_name, email, created_at, updated_at 
    FROM receptionists 
    WHERE id = $1`
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	user := &Receptionist{}
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
	slog.Info("user retrieved", "user", user)
	return user, nil
}

func (s *ReceptionistModel) delete(ctx context.Context, tx *sql.Tx, id int64) error {
	query := `DELETE FROM users WHERE id = $1`
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	_, err := tx.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	return nil
}

func (s *ReceptionistModel) Delete(ctx context.Context, userId int64) error {
	// Implementation for deleting a post
	return withTx(s.DB, ctx, func(tx *sql.Tx) error {
		if err := s.delete(ctx, tx, userId); err != nil {
			return err
		}
		//Normally this should be a soft delete
		if err := s.deleteUserInvitation(ctx, tx, userId); err != nil {
			return err
		}
		return nil
	})
}

// func (s *ReceptionistModel) Create(ctx context.Context, user *Receptionist,) error {
// 	// transaction wrapper
// 	return withTx(s.DB, ctx, func(tx *sql.Tx) error {
// 		// create the user
// 		if err := s.Create(ctx, tx, user); err != nil {
// 			slog.Info("LINE 97", "err", err.Error())
// 			return err
// 		}

// 		return nil

// 	})
// }

func ValidateEmail(v *validator.Validator, email string) {
	v.Check(email != "", "email", "must be provided")
	v.Check(validator.Matches(email, validator.EmailRX), "email", "must be a valid email address")
}
func ValidatePasswordPlaintext(v *validator.Validator, password string) {
	v.Check(password != "", "password", "must be provided")
	v.Check(len(password) >= 8, "password", "must be at least 8 bytes long")
	v.Check(len(password) <= 72, "password", "must not be more than 72 bytes long")
}

func (s *ReceptionistModel) createUserInvitation(ctx context.Context, tx *sql.Tx, token string, exp time.Duration, userID int64) error {
	query := `INSERT INTO user_invitations (token , user_id, expiry) VALUES ($1, $2, $3)`
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	_, err := tx.ExecContext(ctx, query, token, userID, time.Now().Add(exp))
	if err != nil {
		return err
	}
	return nil

}

// func (s *ReceptionistModel) Activate(ctx context.Context, token string) error {
// 	return withTx(s.DB, ctx, func(tx *sql.Tx) error {
// 		// 1. find the user that this token belongs to
// 		user, err := s.getUserFromInvitation(ctx, tx, token)
// 		slog.Info("user", "user", user)
// 		if err != nil {
// 			return err
// 		}

// 		// 2. update the user
// 		user.IsActive = true
// 		if err := s.update(ctx, tx, user); err != nil {
// 			return err
// 		}
// 		// // 3. clean the invitations
// 		// if err := s.deleteUserInvitation(ctx, tx, user.ID); err != nil {
// 		// 	return err
// 		// }
// 		return nil
// 	})
// }

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

func (s *ReceptionistModel) deleteUserInvitation(ctx context.Context, tx *sql.Tx, _ int64) error {
	return nil
}

func (s *ReceptionistModel) GetByEmail(ctx context.Context, email string) (*Receptionist, error) {
	query := `
		SELECT id, first_name, email, password, created_at FROM receptionists
		WHERE email = $1
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	user := &Receptionist{}
	err := s.DB.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.FirstName,
		&user.Email,
		&user.Password.hash,
		&user.CreatedAt,
	)
	fmt.Println(user)
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

// func ValidateUser(v *validator.Validator, user *User) {
// 	v.Check(user.Username != "", "name", "must be provided")
// 	v.Check(len(user.Name) <= 500, "name", "must not be more than 500 bytes long")
// 	// Call the standalone ValidateEmail() helper.
// 	ValidateEmail(v, user.Email)

// 	// If the plaintext password is not nil, call the standalone
// 	// ValidatePasswordPlaintext() helper.
// 	if user.Password.plaintext != nil {
// 		ValidatePasswordPlaintext(v, *user.Password.plaintext)
// 	}

// 	// If the password hash is ever nil, this will be due to a logic error in our
// 	//codebase (probably because we forgot to set a password for the user). It's a
// 	// useful sanity check to include here, but it's not a problem with the data
// 	//provided by the client. So rather than adding an error to the validation map we
// 	// raise a panic instead
// 	if user.Password.hash == nil {
// 		panic("missing password hash for user")
// 	}

// }
