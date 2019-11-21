-- +goose Up

CREATE TABLE snapshot(
    app_id      INTEGER PRIMARY KEY,
    limit_state JSONB
);

-- +goose Down
DROP SCHEMA IF EXISTS gate_service CASCADE;