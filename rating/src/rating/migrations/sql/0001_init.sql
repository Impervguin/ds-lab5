CREATE TABLE rating
(
    id       SERIAL PRIMARY KEY,
    username VARCHAR(80) UNIQUE NOT NULL,
    stars    INT                NOT NULL
        CHECK (stars BETWEEN 1 AND 100)
);
