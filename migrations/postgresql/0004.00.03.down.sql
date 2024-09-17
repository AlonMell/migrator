BEGIN;

DROP TABLE IF EXISTS orders;

DELETE FROM migrations_history
WHERE major_version = '00' AND minor_version = '03' AND file_number = '0004';

COMMIT;