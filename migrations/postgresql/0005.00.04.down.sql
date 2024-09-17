BEGIN;

DROP TABLE IF EXISTS order_items;

DELETE FROM migrations_history
WHERE major_version = '00' AND minor_version = '04' AND file_number = '0005';

COMMIT;