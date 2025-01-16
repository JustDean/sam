-- +goose Up
-- +goose StatementBegin
CREATE TABLE users (
    username VARCHAR(32) PRIMARY KEY,
    password VARCHAR NOT NULL
);
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    valid_through TIMESTAMPTZ NOT NULL,
    username VARCHAR(32) REFERENCES users (username) NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE sessions;
DROP TABLE users;
-- +goose StatementEnd
