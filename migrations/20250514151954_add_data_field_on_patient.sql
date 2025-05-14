-- +goose Up
ALTER TABLE patients
ADD COLUMN data JSONB DEFAULT '{}'::jsonb;

-- +goose Down
ALTER TABLE patients
DROP COLUMN data;