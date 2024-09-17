BEGIN;

DROP TABLE IF EXISTS products;

DELETE FROM migrations_history
WHERE major_version = '00' AND minor_version = '02' AND file_number = '0003';

COMMIT;