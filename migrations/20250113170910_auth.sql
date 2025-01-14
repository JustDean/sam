-- +goose Up
-- +goose StatementBegin
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(32) NOT NULL UNIQUE,
    password VARCHAR NOT NULL
);
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    valid_through TIMESTAMPTZ NOT NULL,
    user_id UUID REFERENCES users (id) NOT NULL
);
CREATE INDEX sessions__valid_through__user_id__index ON sessions (valid_through, user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX sessions__valid_through__user_id__index;
DROP TABLE sessions;
DROP TABLE users;
-- +goose StatementEnd
