-- +goose Up
INSERT INTO library (library_uid, name, city, address)
VALUES ('83575e12-7ce0-48ee-9931-51919ff3c9ee',
        'Библиотека имени 7 Непьющих',
        'Москва',
        '2-я Бауманская ул., д.5, стр.1')
ON CONFLICT (library_uid) DO NOTHING;

INSERT INTO books (book_uid, name, author, genre)
VALUES ('f7cdc58f-2caf-4b15-9727-f89dcc629b27',
        'Краткий курс C++ в 7 томах',
        'Бьерн Страуструп',
        'Научная фантастика')
ON CONFLICT (book_uid) DO NOTHING;

INSERT INTO library_books (library_id, book_id, condition, available_count)
SELECT l.id, b.id, 'EXCELLENT', 1
FROM library l,
     books b
WHERE l.library_uid = '83575e12-7ce0-48ee-9931-51919ff3c9ee'
  AND b.book_uid = 'f7cdc58f-2caf-4b15-9727-f89dcc629b27'
ON CONFLICT (library_id, book_id, condition) DO NOTHING;

-- +goose Down
DELETE
FROM library_books
WHERE library_id IN (SELECT id FROM library WHERE library_uid = '83575e12-7ce0-48ee-9931-51919ff3c9ee')
  AND book_id IN (SELECT id FROM books WHERE book_uid = 'f7cdc58f-2caf-4b15-9727-f89dcc629b27');

DELETE FROM books WHERE book_uid = 'f7cdc58f-2caf-4b15-9727-f89dcc629b27';
DELETE FROM library WHERE library_uid = '83575e12-7ce0-48ee-9931-51919ff3c9ee';
