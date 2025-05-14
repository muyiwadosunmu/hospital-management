-- +goose Up
-- Create doctors table
CREATE TABLE
    IF NOT EXISTS doctors (
        id BIGSERIAL PRIMARY KEY,
        first_name VARCHAR(100) NOT NULL,
        last_name VARCHAR(100) NOT NULL,
        email VARCHAR(255) NOT NULL UNIQUE,
        password BYTEA NOT NULL,
        created_at TIMESTAMP
        WITH
            TIME ZONE NOT NULL DEFAULT NOW (),
            updated_at TIMESTAMP
        WITH
            TIME ZONE NOT NULL DEFAULT NOW ()
    );

-- Create indexes
CREATE INDEX idx_doctors_email ON doctors (email);

CREATE INDEX idx_doctors_name ON doctors (first_name, last_name);

-- +goose Down
DROP TRIGGER IF EXISTS update_doctors_updated_at ON doctors;

DROP TABLE IF EXISTS doctors;