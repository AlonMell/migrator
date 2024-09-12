BEGIN;

CREATE TABLE subscriptions
(
    id                   UUID PRIMARY KEY   DEFAULT gen_random_uuid(),
    created_at           TIMESTAMP          DEFAULT CURRENT_TIMESTAMP,
    updated_at           TIMESTAMP          DEFAULT CURRENT_TIMESTAMP,
    user_id              UUID      NOT NULL REFERENCES users (id),
    subscription_plan_id UUID      NOT NULL REFERENCES subscription_plans (id),
    status               BOOLEAN   NOT NULL DEFAULT FALSE,
    start_date           TIMESTAMP          DEFAULT CURRENT_TIMESTAMP,
    end_date             TIMESTAMP NOT NULL
);

CREATE TABLE subscription_plans
(
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at  TIMESTAMP        DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP        DEFAULT CURRENT_TIMESTAMP,
    service_id  UUID REFERENCES services (id),
    description VARCHAR(100),
    price       NUMERIC(10, 2) NOT NULL CHECK (price >= 0),
    duration    INTERVAL       NOT NULL CHECK (duration > '0')
);

CREATE TABLE services
(
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at  TIMESTAMP        DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP        DEFAULT CURRENT_TIMESTAMP,
    name        VARCHAR(50) NOT NULL,
    description VARCHAR(100)
);

INSERT INTO migrations_history(major_Version, minor_version, file_number, comment)
VALUES ('00', '01', '0001', 'add subscription entities');

COMMIT;