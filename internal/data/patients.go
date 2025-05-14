package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

type Patient struct {
	ID        int64         `json:"id"`
	FirstName string        `json:"firstName"`
	LastName  string        `json:"lastName"`
	Email     string        `json:"email"`
	Password  password      `json:"-"`
	CreatedAt time.Time     `json:"createdAt"`
	UpdatedAt time.Time     `json:"-"`
	AddedBy   *Receptionist `json:"receptionist"`
	Data      interface{}   `json:"data"`
	Version   int64         `json:"version"`
}

type PatientModel struct {
	DB *sql.DB
}

func (s *PatientModel) CreatePatient(ctx context.Context, user *Patient) error {
	query := `INSERT INTO patients (first_name,last_name, email, password, receptionist_id) 
	VALUES ($1, $2, $3, $4, $5) RETURNING id, first_name, last_name, created_at`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()
	fmt.Println(user)

	err := s.DB.QueryRowContext(ctx, query, user.FirstName,
		user.LastName, user.Email, user.Password.hash,
		user.AddedBy.ID).
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

func (m *PatientModel) Get(ctx context.Context, firstName, lastName string, filters Filters) ([]*Patient, Metadata, error) {
	query := fmt.Sprintf(`
	SELECT count(*) OVER(), id, email, created_at, first_name, last_name, version
	FROM patients
	WHERE (to_tsvector('simple', first_name) @@ plainto_tsquery('simple', $1) OR $1 = '')
	AND (to_tsvector('simple', last_name) @@ plainto_tsquery('simple', $2) OR $2 = '')
	ORDER BY %s %s, id ASC
	LIMIT $3 OFFSET $4`, filters.sortColumn(), filters.sortDirection())

	args := []interface{}{firstName, lastName, filters.limit(), filters.offset()}

	// Use QueryContext() to execute the query. This returns a sql.Rows result set
	// containing the result.
	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}

	// Importantly, defer a call to rows.Close() to ensure that the result set is closed
	// before GetAll() returns.
	defer rows.Close()
	totalRecords := 0
	// Initialize an empty slice to hold the movie data.
	patients := []*Patient{}
	// Use rows.Next to iterate through the rows in the result set.
	for rows.Next() {
		// Initialize an empty Movie struct to hold the data for an individual movie.
		var patient Patient
		// Scan the values from the row into the Movie struct. Again, note that we're
		// using the pq.Array() adapter on the genres field here.
		err := rows.Scan(
			&totalRecords,
			&patient.ID,
			&patient.Email,
			&patient.CreatedAt,
			&patient.FirstName,
			&patient.LastName,
			&patient.Version,
		)
		if err != nil {
			return nil, Metadata{}, err
		}
		// Add the Movie struct to the slice.
		patients = append(patients, &patient)
	}
	// When the rows.Next() loop has finished, call rows.Err() to retrieve any error
	// that was encountered during the iteration.
	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err // Update this to return an empty Metadata struct.
	}

	// Generate a Metadata struct, passing in the total record count and pagination
	// parameters from the client.
	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)

	// If everything went OK, then return the slice of movies.
	return patients, metadata, nil

}

func (m *PatientModel) UpdatePatient(ctx context.Context, patient *Patient) error {
	if patient.ID < 1 {
		return ErrRecordNotFound
	}
	query := `UPDATE patients
			 SET first_name = $1, last_name = $2, version = version + 1 
			 WHERE id = $3 AND VERSION = $4 
			 RETURNING version`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()
	err := m.DB.QueryRowContext(ctx, query, patient.FirstName, patient.LastName, patient.ID, patient.Version).
		Scan(&patient.Version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrRecordNotFound
		case errors.Is(err, ErrEditConflict):
			return ErrEditConflict
		default:
			return err

		}
	}

	return nil
}

func (m *PatientModel) UpdatePatientByDoc(ctx context.Context, patient *Patient) error {
	if patient.ID < 1 {
		return ErrRecordNotFound
	}

	// Convert the data field to JSON bytes
	dataJSON, err := json.Marshal(patient.Data)
	if err != nil {
		return fmt.Errorf("error marshaling patient data: %w", err)
	}

	query := `UPDATE patients
             SET first_name = $1, last_name = $2, version = version + 1, data = $3
             WHERE id = $4 AND VERSION = $5 
             RETURNING version`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	err = m.DB.QueryRowContext(ctx, query,
		patient.FirstName,
		patient.LastName,
		dataJSON, // Use the marshaled JSON
		patient.ID,
		patient.Version).
		Scan(&patient.Version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrRecordNotFound
		case errors.Is(err, ErrEditConflict):
			return ErrEditConflict
		default:
			return err
		}
	}

	return nil
}

func (s *PatientModel) GetDocPatientById(ctx context.Context, id int64) (*Patient, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}
	query := `SELECT id, first_name, last_name, email, created_at,
    updated_at, version, data
    FROM patients 
    WHERE id = $1`
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	user := &Patient{}
	var dataJSON []byte
	err := s.DB.QueryRowContext(ctx, query, id).
		Scan(&user.ID,
			&user.FirstName,
			&user.LastName,
			&user.Email,
			&user.CreatedAt,
			&user.UpdatedAt,
			&user.Version,
			&dataJSON)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	// Unmarshal the JSON data into a map
	if dataJSON != nil {
		var data map[string]interface{}
		if err := json.Unmarshal(dataJSON, &data); err != nil {
			return nil, fmt.Errorf("error unmarshaling patient data: %w", err)
		}
		user.Data = data
	}

	slog.Info("user retrieved", "user", user)
	return user, nil
}

func (s *PatientModel) GetPatientById(ctx context.Context, id int64) (*Patient, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}
	query := `SELECT id, first_name, last_name, email, created_at,
    updated_at, version
    FROM patients 
    WHERE id = $1`
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	user := &Patient{}
	err := s.DB.QueryRowContext(ctx, query, id).
		Scan(&user.ID,
			&user.FirstName,
			&user.LastName,
			&user.Email,
			&user.CreatedAt,
			&user.UpdatedAt,
			&user.Version)
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

func (m *PatientModel) Delete(ctx context.Context, id int64) error {
	if id < 1 {
		return ErrRecordNotFound
	}
	query := `DELETE FROM patients WHERE id = $1`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()
	_, err := m.DB.ExecContext(ctx, query, id)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrRecordNotFound
		default:
			return err
		}
	}
	return nil
}
