CREATE TABLE reservation
(
    id                SERIAL PRIMARY KEY,
    reservation_uid   uuid UNIQUE NOT NULL,
    username          VARCHAR(80) NOT NULL,
    book_uid          uuid        NOT NULL,
    library_uid       uuid        NOT NULL,
    status            VARCHAR(20) NOT NULL
        CHECK (status IN ('RENTED', 'RETURNED', 'EXPIRED')),
    start_date        DATE        NOT NULL,
    till_date         DATE        NOT NULL,
    condition_at_rent VARCHAR(20) NOT NULL
        CHECK (condition_at_rent IN ('EXCELLENT', 'GOOD', 'BAD'))
);

CREATE INDEX reservation_username_status_idx ON reservation (username, status);
