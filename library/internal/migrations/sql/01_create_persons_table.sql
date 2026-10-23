-- +goose Up
CREATE TABLE library
(
    id          SERIAL PRIMARY KEY,
    library_uid uuid UNIQUE  NOT NULL,
    name        VARCHAR(80)  NOT NULL,
    city        VARCHAR(255) NOT NULL,
    address     VARCHAR(255) NOT NULL
);

CREATE INDEX library_city_idx ON library (city);

CREATE TABLE books
(
    id       SERIAL PRIMARY KEY,
    book_uid uuid UNIQUE  NOT NULL,
    name     VARCHAR(255) NOT NULL,
    author   VARCHAR(255),
    genre    VARCHAR(255)
);

CREATE TABLE library_books
(
    book_id         INT         NOT NULL REFERENCES books (id),
    library_id      INT         NOT NULL REFERENCES library (id),
    condition       VARCHAR(20) NOT NULL DEFAULT 'EXCELLENT'
        CHECK (condition IN ('EXCELLENT', 'GOOD', 'BAD')),
    available_count INT         NOT NULL DEFAULT 0
        CHECK (available_count >= 0),
    PRIMARY KEY (library_id, book_id, condition)
);

-- +goose Down
DROP TABLE IF EXISTS library_books;
DROP TABLE IF EXISTS books;
DROP TABLE IF EXISTS library;
