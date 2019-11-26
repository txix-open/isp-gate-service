-- +goose Up

ALTER TABLE snapshot
    ADD version BIGINT NOT NULL DEFAULT 0;

-- +goose Down