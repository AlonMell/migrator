BEGIN;

DROP TABLE IF EXISTS subscriptions;
DROP TABLE IF EXISTS subscription_plans;
DROP TABLE IF EXISTS services;

DELETE FROM migrations_history
WHERE major_version = '00' AND minor_version = '01' AND file_number = '0002';

COMMIT;