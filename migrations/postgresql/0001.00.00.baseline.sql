BEGIN;

CREATE TABLE IF NOT EXISTS migrations_history

(
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    date_applied  TIMESTAMP        DEFAULT CURRENT_TIMESTAMP,
    major_version VARCHAR(2),
    minor_version VARCHAR(2),
    file_number   VARCHAR(4),
    comment       TEXT
);

INSERT INTO migrations_history(major_Version, minor_version, file_number, comment)
VALUES ('00', '00', '0001', 'baseline');

CREATE TABLE IF NOT EXISTS users
(
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at    TIMESTAMP        DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP        DEFAULT CURRENT_TIMESTAMP,
    email         VARCHAR(100) NOT NULL UNIQUE CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$'),
    password_hash VARCHAR(100) NOT NULL,
    --phone         VARCHAR(20) UNIQUE CHECK (phone ~ '^\+\d{1,11}$'),
    is_active     BOOLEAN          DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS roles
(
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at  TIMESTAMP        DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP        DEFAULT CURRENT_TIMESTAMP,
    name        VARCHAR(50) UNIQUE NOT NULL,
    description TEXT
);

CREATE TABLE IF NOT EXISTS permissions
(
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at  TIMESTAMP        DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP        DEFAULT CURRENT_TIMESTAMP,
    name        VARCHAR(50) UNIQUE NOT NULL,
    description TEXT
);

CREATE TABLE IF NOT EXISTS user_roles
(
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles (id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

CREATE TABLE IF NOT EXISTS role_permissions
(
    role_id       UUID NOT NULL REFERENCES roles (id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions (id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

CREATE INDEX IF NOT EXISTS idx_email ON users (email);

CREATE OR REPLACE FUNCTION update_timestamp()
    RETURNS TRIGGER AS
$$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER update_users_timestamp
    BEFORE UPDATE
    ON users
    FOR EACH ROW
EXECUTE FUNCTION update_timestamp();

CREATE OR REPLACE TRIGGER update_roles_timestamp
    BEFORE UPDATE
    ON roles
    FOR EACH ROW
EXECUTE FUNCTION update_timestamp();

CREATE OR REPLACE TRIGGER update_permissions_timestamp
    BEFORE UPDATE
    ON permissions
    FOR EACH ROW
EXECUTE FUNCTION update_timestamp();

SELECT major_version, minor_version
FROM migrations_history
ORDER BY date_applied DESC
LIMIT 1;

COMMIT;



