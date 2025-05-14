-- +goose Up
-- +goose StatementBegin
SELECT
    'up SQL query';

ALTER TABLE patients
ADD COLUMN version INT default 0;

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
SELECT
    'down SQL query';

ALTER TABLE posts
DROP COLUMN version;

-- +goose StatementEnd