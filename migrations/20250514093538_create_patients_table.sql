-- +goose Up
CREATE TABLE
    IF NOT EXISTS patients (
        id BIGSERIAL PRIMARY KEY,
        first_name VARCHAR(100) NOT NULL,
        last_name VARCHAR(100) NOT NULL,
        email VARCHAR(255) NOT NULL UNIQUE,
        password BYTEA NOT NULL,
        receptionist_id BIGINT NOT NULL REFERENCES receptionists (id),
        created_at TIMESTAMP
        WITH
            TIME ZONE NOT NULL DEFAULT NOW (),
            updated_at TIMESTAMP
        WITH
            TIME ZONE NOT NULL DEFAULT NOW ()
    );

-- Create indexes
CREATE INDEX idx_patients_email ON patients (email);

CREATE INDEX idx_patients_name ON patients (first_name, last_name);

-- +goose Down
DROP TABLE IF EXISTS patients;