-- +goose Up
-- Create receptionists table
CREATE TABLE
    IF NOT EXISTS receptionists (
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
CREATE INDEX idx_receptionists_email ON receptionists (email);

CREATE INDEX idx_receptionists_name ON receptionists (first_name, last_name);

-- +goose Down
DROP TRIGGER IF EXISTS update_receptionists_updated_at ON receptionists;

DROP FUNCTION IF EXISTS update_updated_at_column ();

DROP TABLE IF EXISTS receptionists;