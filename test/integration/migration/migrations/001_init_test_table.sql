-- +goose Up
CREATE TABLE IF NOT EXISTS gorm
(
    val text NOT NULL
);

CREATE TABLE IF NOT EXISTS pgx
(
    val text NOT NULL
);

CREATE TABLE IF NOT EXISTS sqlx
(
    val text NOT NULL
);

CREATE TABLE IF NOT EXISTS stdlib
(
    val text NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS gorm;
DROP TABLE IF EXISTS pgx;
DROP TABLE IF EXISTS sqlx;
DROP TABLE IF EXISTS stdlib;
