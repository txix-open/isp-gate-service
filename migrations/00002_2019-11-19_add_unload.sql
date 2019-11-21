-- +goose Up

CREATE TABLE requests
(
    id         SERIAL8 PRIMARY KEY,
    app_id     INT4         NOT NULL,
    method     VARCHAR(255) NOT NULL,
    created_at TIMESTAMP    NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE requests;
