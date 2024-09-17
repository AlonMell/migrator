BEGIN;

CREATE TABLE orders
(
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at  TIMESTAMP        DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP        DEFAULT CURRENT_TIMESTAMP,
    user_id     UUID NOT NULL REFERENCES users (id),
    total_price NUMERIC(10, 2) NOT NULL CHECK (total_price >= 0)
);

INSERT INTO migrations_history(major_version, minor_version, file_number, comment)
VALUES ('00', '03', '0004', 'add orders table');

COMMIT;