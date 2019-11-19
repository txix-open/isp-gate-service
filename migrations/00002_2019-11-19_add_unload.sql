-- +goose Up

CREATE TABLE unload
(
    id         SERIAL8 PRIMARY KEY,
    app_id     INTEGER      NOT NULL,
    method     VARCHAR(255) NOT NULL,
    created_at TIMESTAMP    NOT NULL
);

-- +goose Down
DROP TABLE unload;